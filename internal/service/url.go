package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// URLService defines the interface for the URL service
type URLService interface {
	CreateShortURL(ctx context.Context, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	GetStorage() storage.URLStorage
	CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	CheckConnection(ctx context.Context) error
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
	Save(ctx context.Context, shortURL, originalURL string) error
	SaveBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
}

// URLServiceImpl implements the URLService
type URLServiceImpl struct {
	storage storage.URLStorage
	config  *config.Config
	logger  *zap.Logger
}

// NewURLService creates a new instance of URLService, choosing storage
// depending on the configuration with priority: PostgreSQL -> File -> Memory.
func NewURLService(cfg *config.Config, logger *zap.Logger) (URLService, error) {
	var store storage.URLStorage
	var err error

	// 1. Try to use PostgreSQL, if DSN is specified
	if cfg.DatabaseDSN != "" {
		log.Println("Using PostgreSQL storage:", cfg.DatabaseDSN)
		store, err = storage.NewPostgresStorage(cfg.DatabaseDSN, logger)
		if err != nil {
			// Log the error but don't exit, as we can switch to file storage
			log.Printf("PostgreSQL storage initialization error: %v. Switching to file storage.", err)
		} else {
			// Successfully created PostgresStorage, use it
			return &URLServiceImpl{
				storage: store,
				config:  cfg,
				logger:  logger,
			}, nil
		}
	}

	// 2. Try to use File Storage, if path is specified and Postgres failed or not specified
	if cfg.FileStoragePath != "" {
		log.Println("Using file storage:", cfg.FileStoragePath)
		store, err = storage.NewFileStorage(cfg.FileStoragePath, logger)
		if err != nil {
			// Log the error but don't exit, as we can switch to in-memory storage
			log.Printf("File storage initialization error: %v. Switching to in-memory storage.", err)
		} else {
			// Successfully created FileStorage, use it
			return &URLServiceImpl{
				storage: store,
				config:  cfg,
				logger:  logger,
			}, nil
		}
	}

	// 3. Use in-memory storage as a fallback option
	log.Println("Using in-memory storage.")
	store = storage.NewMemoryStorage(logger) // Передаем логгер в конструктор

	return &URLServiceImpl{
		storage: store,
		config:  cfg,
		logger:  logger,
	}, nil // There should be no errors here, as MemoryStorage always succeeds
}

// CreateShortURL создает короткий URL из оригинального
func (s *URLServiceImpl) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	if originalURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Проверяем формат URL
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return "", storage.ErrInvalidURL
	}

	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return "", fmt.Errorf("user_id not found in context")
	}

	// Проверяем, существует ли уже такой URL
	if shortURL, err := s.storage.GetShortURLByOriginal(ctx, originalURL); err == nil {
		return shortURL, storage.ErrOriginalURLConflict
	}

	// Генерируем короткий URL
	hash := sha256.Sum256([]byte(originalURL))
	shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]

	// Сохраняем URL с привязкой к пользователю
	if err := s.storage.SaveUserURL(ctx, userID, shortURL, originalURL); err != nil {
		return "", fmt.Errorf("error saving URL: %w", err)
	}

	return shortURL, nil
}

// GetOriginalURL gets the original URL from the short
func (s *URLServiceImpl) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", fmt.Errorf("empty short URL")
	}

	originalURL, err := s.storage.Get(ctx, shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("URL not found for shortID: %s", shortURL)
		} else {
			log.Printf("Error getting URL for shortID %s: %v", shortURL, err)
		}
		return "", err // Return error (including ErrURLNotFound)
	}
	return originalURL, nil
}

// CreateShortURLsBatch creates short URLs for a batch and saves them
func (s *URLServiceImpl) CreateShortURLsBatch(ctx context.Context, reqBatch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	if len(reqBatch) == 0 {
		return []models.BatchResponseEntry{}, nil // Return empty slice if input is empty
	}

	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user_id not found in context")
	}

	storageBatch := make([]storage.BatchEntry, 0, len(reqBatch))
	respBatch := make([]models.BatchResponseEntry, 0, len(reqBatch))

	for _, reqEntry := range reqBatch {
		originalURL := reqEntry.OriginalURL
		if originalURL == "" {
			// Skip empty URLs or return error?
			// According to the task, it's not specified, but logically we should skip or return an error for the whole batch.
			// For now, we'll skip and log.
			log.Printf("Skipped empty URL in batch for correlation_id: %s", reqEntry.CorrelationID)
			continue
		}
		// TODO: Add URL validation, like in CreateShortURL?
		// _, err := url.ParseRequestURI(originalURL)
		// if err != nil { ... }

		// Check if URL already exists
		existingShortURL, err := s.storage.GetShortURLByOriginal(ctx, originalURL)
		if err == nil {
			// URL already exists, use existing short URL
			respBatch = append(respBatch, models.BatchResponseEntry{
				CorrelationID: reqEntry.CorrelationID,
				ShortURL:      existingShortURL,
			})
			continue
		}

		// Generate shortURL (same logic as in CreateShortURL)
		hash := sha256.Sum256([]byte(originalURL))
		shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]
		fullShortURL := s.config.BaseURL + "/" + shortURL // Form full URL for response

		// Add to batch for saving to storage
		storageBatch = append(storageBatch, storage.BatchEntry{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
			UserID:      userID,
		})

		// Add to batch for response
		respBatch = append(respBatch, models.BatchResponseEntry{
			CorrelationID: reqEntry.CorrelationID,
			ShortURL:      fullShortURL,
		})
	}

	// If after filtering empty URLs batch for storage became empty
	if len(storageBatch) == 0 {
		log.Println("Batch for saving is empty after processing input data.")
		// Return empty respBatch, since no valid URLs were found
		return []models.BatchResponseEntry{}, errors.New("all URLs in the batch were invalid or empty")
	}

	// Save entire batch to storage
	err := s.storage.SaveBatch(ctx, storageBatch)
	if err != nil {
		log.Printf("Error saving URL batch: %v", err)
		return nil, fmt.Errorf("error saving batch: %w", err) // Return error
	}

	// Return result
	return respBatch, nil
}

// GetStorage returns the URL storage
func (s *URLServiceImpl) GetStorage() storage.URLStorage {
	return s.storage
}

// CheckConnection проверяет соединение с хранилищем
func (s *URLServiceImpl) CheckConnection(ctx context.Context) error {
	// Проверяем, реализует ли хранилище интерфейс DatabaseChecker
	if checker, ok := s.storage.(storage.DatabaseChecker); ok {
		return checker.CheckConnection(ctx)
	}
	// Если хранилище не поддерживает проверку соединения, считаем что оно доступно
	return nil
}

// GetUserURLs получает все URL пользователя
func (s *URLServiceImpl) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	return s.storage.GetUserURLs(ctx, userID)
}

// Save сохраняет URL в хранилище
func (s *URLServiceImpl) Save(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	_, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	return s.storage.Save(ctx, shortURL, originalURL)
}

// SaveBatch сохраняет пакет URL
func (s *URLServiceImpl) SaveBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	// Получаем userID из контекста
	_, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user_id not found in context")
	}

	// ... existing code ...
	return nil, nil
}

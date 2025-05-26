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

// CreateShortURL creates a short URL from the original
func (s *URLServiceImpl) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	if originalURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Check URL validity
	_, err := url.ParseRequestURI(originalURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format")
	}

	// Check if URL already exists
	existingShortURL, err := s.storage.GetShortURLByOriginal(ctx, originalURL)
	if err == nil {
		// URL already exists, return existing short URL
		s.logger.Info("URL already exists, returning existing short URL",
			zap.String("original_url", originalURL),
			zap.String("short_url", existingShortURL))
		return existingShortURL, storage.ErrOriginalURLConflict
	}

	hash := sha256.Sum256([]byte(originalURL))
	shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]

	err = s.storage.Save(ctx, shortURL, originalURL)
	if err != nil {
		// Check if the error is due to a conflict with the original URL
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			// If URL already exists, get existing shortURL
			log.Printf("Conflict: Original URL '%s' already exists. Getting existing short URL.", originalURL)
			existingShortURL, getErr := s.storage.GetShortURLByOriginal(ctx, originalURL)
			if getErr != nil {
				// This situation should not occur if Save returned a conflict,
				// but we handle it on the safe side
				log.Printf("Critical error: failed to get short URL for existing original URL '%s': %v", originalURL, getErr)
				return "", fmt.Errorf("error getting existing short URL: %w", getErr)
			}
			// Return existing shortURL and conflict error for handling in the handler
			return existingShortURL, storage.ErrOriginalURLConflict
		}

		// For other saving errors, just log and return
		log.Printf("Error saving URL: %v", err)
		return "", err
	}

	// If there's no error, return new shortURL
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

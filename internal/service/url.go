package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// URLService определяет интерфейс для работы с URL
type URLService interface {
	// CreateShortURL создает короткий URL для оригинального URL
	CreateShortURL(ctx context.Context, originalURL string) (string, error)
	// GetOriginalURL возвращает оригинальный URL по короткому URL
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	// CreateShortURLsBatch создает короткие URL для пакета оригинальных URL
	CreateShortURLsBatch(ctx context.Context, batch []URLBatchItem) ([]URLBatchResult, error)
	// GetUserURLs возвращает список URL пользователя
	GetUserURLs(ctx context.Context) ([]models.UserURL, error)
	// Ping проверяет соединение с хранилищем
	Ping(ctx context.Context) error
}

// UserURL представляет URL пользователя
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// URLBatchItem представляет элемент пакетного запроса для создания коротких URL
type URLBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// URLBatchResult представляет результат пакетного создания коротких URL
type URLBatchResult struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// URLServiceImpl реализует бизнес-логику для работы с URL
type URLServiceImpl struct {
	storage storage.URLStorage
	logger  *zap.Logger
	config  *config.Config
}

// NewURLService создает новый экземпляр URLService
func NewURLService(cfg *config.Config, logger *zap.Logger) (URLService, error) {
	var s storage.URLStorage
	var err error

	storageType := "memory" // По умолчанию используем память
	if cfg.DatabaseDSN != "" {
		storageType = "postgres"
	} else if cfg.FileStoragePath != "" {
		storageType = "file"
	}

	switch storageType {
	case "memory":
		s = storage.NewMemoryStorage(logger)
	case "file":
		s, err = storage.NewFileStorage(cfg.FileStoragePath, logger)
	case "postgres":
		s, err = storage.NewPostgresStorage(context.Background(), cfg.DatabaseDSN, logger)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &URLServiceImpl{
		storage: s,
		logger:  logger,
		config:  cfg,
	}, nil
}

// CreateShortURL создает короткий URL для оригинального URL
func (s *URLServiceImpl) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	// Проверяем, что URL валидный
	if !isValidURL(originalURL) {
		return "", storage.ErrInvalidURL
	}

	// Генерируем короткий URL
	shortURL := generateShortURL(originalURL)

	// Сохраняем URL
	if err := s.storage.SaveURL(ctx, shortURL, originalURL); err != nil {
		if err == storage.ErrURLAlreadyExists {
			// Если URL уже существует, получаем его короткую версию
			existingShortURL, err := s.storage.GetShortURL(ctx, originalURL)
			if err != nil {
				return "", fmt.Errorf("failed to get existing short URL: %w", err)
			}
			return existingShortURL, nil
		}
		return "", fmt.Errorf("failed to save URL: %w", err)
	}

	return shortURL, nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *URLServiceImpl) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	return s.storage.GetOriginalURL(ctx, shortURL)
}

// GetUserURLs возвращает список URL пользователя
func (s *URLServiceImpl) GetUserURLs(ctx context.Context) ([]models.UserURL, error) {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user_id not found in context")
	}

	return s.storage.GetUserURLs(ctx, userID)
}

// CreateShortURLsBatch создает короткие URL для пакета оригинальных URL
func (s *URLServiceImpl) CreateShortURLsBatch(ctx context.Context, batch []URLBatchItem) ([]URLBatchResult, error) {
	if len(batch) == 0 {
		return nil, nil
	}

	result := make([]URLBatchResult, len(batch))
	for i, item := range batch {
		if !isValidURL(item.OriginalURL) {
			return nil, storage.ErrInvalidURL
		}

		shortURL, err := s.CreateShortURL(ctx, item.OriginalURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create short URL for item %s: %w", item.CorrelationID, err)
		}

		result[i] = URLBatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		}
	}

	return result, nil
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

// Save сохраняет URL в хранилище
func (s *URLServiceImpl) Save(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	_, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	return s.storage.SaveURL(ctx, shortURL, originalURL)
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

// Ping проверяет соединение с хранилищем
func (s *URLServiceImpl) Ping(ctx context.Context) error {
	// Проверяем, реализует ли хранилище интерфейс DatabaseChecker
	if checker, ok := s.storage.(storage.DatabaseChecker); ok {
		return checker.CheckConnection(ctx)
	}
	// Если хранилище не поддерживает проверку соединения, считаем что оно доступно
	return nil
}

// isValidURL проверяет, является ли строка валидным URL
func isValidURL(rawURL string) bool {
	// Добавляем схему, если её нет
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	// Парсим URL
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	// Проверяем, что есть хост
	return u.Host != ""
}

// generateShortURL генерирует короткий URL на основе оригинального URL
func generateShortURL(originalURL string) string {
	// Создаем хеш URL
	hash := sha256.Sum256([]byte(originalURL))

	// Кодируем хеш в base64
	encoded := base64.URLEncoding.EncodeToString(hash[:])

	// Берем первые 8 символов
	return encoded[:8]
}

// Close закрывает соединение с хранилищем
func (s *URLServiceImpl) Close() error {
	return s.storage.Close()
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// BatchRequest представляет запрос на создание короткого URL в пакетном режиме
type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponse представляет результат создания короткого URL в пакетном режиме
type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// URLService определяет интерфейс для работы с URL
type URLService interface {
	// CreateShortURL создает короткий URL для оригинального URL
	CreateShortURL(ctx context.Context, originalURL string, userID string) (string, error)
	// GetOriginalURL возвращает оригинальный URL по короткому URL
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	// CreateShortURLsBatch создает короткие URL для пакета оригинальных URL
	// CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry, userID string) ([]models.BatchResponseEntry, error)
	// GetUserURLs возвращает список URL пользователя
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
	// Ping проверяет соединение с хранилищем
	Ping(ctx context.Context) error
	// GetStorage() storage.URLStorage // Этот метод, похоже, не используется широко и может быть удален, если не нужен
	Close() error // Добавим метод Close для корректного завершения работы с хранилищем
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

var ErrUnauthorized = errors.New("user_id not found in context")

// NewURLService создает новый экземпляр URLService
func NewURLService(cfg *config.Config, logger *zap.Logger) (URLService, error) {
	logger.Info("NewURLService called")
	var s storage.URLStorage
	var err error

	storageType := "memory" // По умолчанию используем память
	if cfg.DatabaseDSN != "" {
		logger.Info("Using postgres storage", zap.String("dsn", cfg.DatabaseDSN))
		storageType = "postgres"
	} else if cfg.FileStoragePath != "" {
		logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
		storageType = "file"
	} else {
		logger.Info("Using memory storage")
	}

	switch storageType {
	case "memory":
		s = storage.NewMemoryStorage(logger)
	case "file":
		s, err = storage.NewFileStorage(cfg.FileStoragePath, logger)
	case "postgres":
		s, err = storage.NewPostgresStorage(context.Background(), cfg.DatabaseDSN, logger) // Передаем context.Background()
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}

	if err != nil {
		logger.Error("Failed to initialize storage", zap.String("type", storageType), zap.Error(err))
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	logger.Info("Storage initialized successfully", zap.String("type", storageType))

	return &URLServiceImpl{
		storage: s,
		logger:  logger,
		config:  cfg,
	}, nil
}

// CreateShortURL создает короткий URL для заданного оригинального URL
func (s *URLServiceImpl) CreateShortURL(ctx context.Context, originalURL string, userID string) (string, error) {
	s.logger.Debug("CreateShortURL called", zap.String("originalURL", originalURL), zap.String("userID", userID))
	if originalURL == "" {
		s.logger.Warn("CreateShortURL: originalURL is empty")
		return "", storage.ErrInvalidURL
	}

	// Валидация URL (можно добавить более строгую)
	parsedURL, err := url.Parse(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		s.logger.Warn("CreateShortURL: invalid originalURL format", zap.String("originalURL", originalURL), zap.Error(err))
		return "", storage.ErrInvalidURL
	}

	shortURL := generateShortURL(originalURL)
	s.logger.Debug("Generated shortURL", zap.String("shortURL", shortURL))

	err = s.storage.SaveURL(ctx, shortURL, originalURL) // userID теперь должен обрабатываться внутри SaveURL, если хранилище это требует
	if err != nil {
		if errors.Is(err, storage.ErrURLAlreadyExists) {
			s.logger.Info("URL already exists, returning existing shortURL", zap.String("originalURL", originalURL), zap.String("shortURL", shortURL))
			return shortURL, nil // Возвращаем существующий shortURL и nil в качестве ошибки, как требует тест и логика хендлера
		}
		s.logger.Error("Failed to save URL to storage", zap.Error(err))
		return "", fmt.Errorf("failed to save URL: %w", err)
	}
	s.logger.Info("URL saved successfully", zap.String("shortURL", shortURL), zap.String("originalURL", originalURL))
	return shortURL, nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *URLServiceImpl) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	s.logger.Debug("GetOriginalURL called", zap.String("shortURL", shortURL))
	if shortURL == "" || shortURL == "." { // Добавил проверку на "."
		s.logger.Warn("GetOriginalURL: shortURL is empty or invalid")
		return "", storage.ErrInvalidURL // Используем кастомную ошибку
	}

	shortID := path.Base(shortURL)       // Убедимся, что это только ID
	if shortID == "" || shortID == "." { // Дополнительная проверка
		s.logger.Warn("GetOriginalURL: extracted shortID is empty or invalid", zap.String("shortURL", shortURL))
		return "", storage.ErrInvalidURL
	}

	originalURL, err := s.storage.GetOriginalURL(ctx, shortID)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			s.logger.Info("Original URL not found for shortID", zap.String("shortID", shortID))
			return "", storage.ErrURLNotFound
		}
		s.logger.Error("Failed to get original URL from storage", zap.String("shortID", shortID), zap.Error(err))
		return "", fmt.Errorf("failed to get original URL: %w", err)
	}
	s.logger.Debug("Original URL retrieved", zap.String("originalURL", originalURL))
	return originalURL, nil
}

// GetUserURLs возвращает список URL пользователя
func (s *URLServiceImpl) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.logger.Debug("GetUserURLs called", zap.String("userID", userID))
	if userID == "" {
		// В зависимости от требований, можно возвращать ошибку или пустой список
		s.logger.Warn("GetUserURLs: userID is empty")
		return nil, ErrUnauthorized // или return []models.UserURL{}, nil
	}
	// userID извлекается из контекста в хендлере и передается сюда
	// userID, ok := ctx.Value(middleware.UserIDKey).(string)
	// if !ok || userID == "" {
	// 	s.logger.Error("GetUserURLs: userID not found in context or is empty")
	// 	return nil, ErrUnauthorized
	// }

	userURLs, err := s.storage.GetUserURLs(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user URLs from storage", zap.String("userID", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to get user URLs: %w", err)
	}
	return userURLs, nil
}

// CreateShortURLsBatch создает короткие URL для пакета оригинальных URL
// func (s *URLServiceImpl) CreateShortURLsBatch(ctx context.Context, batchReq []models.BatchRequestEntry, userID string) ([]models.BatchResponseEntry, error) {
// 	s.logger.Debug("CreateShortURLsBatch called", zap.Int("batch_size", len(batchReq)), zap.String("userID", userID))
// 	if len(batchReq) == 0 {
// 		s.logger.Warn("CreateShortURLsBatch: batch is empty")
// 		return nil, fmt.Errorf("empty batch")
// 	}

// 	// Трансформируем в формат для хранилища, если это необходимо, или передаем как есть
// 	// В данном случае, s.storage.SaveURLsBatch ожидает []storage.URLItem
// 	urlItems := make([]storage.URLItem, len(batchReq))
// 	responseEntries := make([]models.BatchResponseEntry, len(batchReq))

// 	for i, item := range batchReq {
// 		if item.OriginalURL == "" {
// 			s.logger.Warn("CreateShortURLsBatch: item has empty OriginalURL", zap.String("correlationID", item.CorrelationID))
// 			// Можно вернуть ошибку или пропустить этот элемент, в зависимости от требований
// 			// Для примера, вернем ошибку на весь батч
// 			return nil, fmt.Errorf("item with CorrelationID '%s' has empty OriginalURL", item.CorrelationID)
// 		}
// 		shortURL := generateShortURL(item.OriginalURL)
// 		urlItems[i] = storage.URLItem{ShortURL: shortURL, OriginalURL: item.OriginalURL}
// 		responseEntries[i] = models.BatchResponseEntry{
// 			CorrelationID: item.CorrelationID,
// 			ShortURL:      fmt.Sprintf("%s/%s", s.config.BaseURL, shortURL), // Формируем полный URL для ответа
// 		}
// 	}

// 	err := s.storage.SaveURLsBatch(ctx, urlItems) // userID должен обрабатываться внутри SaveURLsBatch, если нужно
// 	if err != nil {
// 		s.logger.Error("Failed to save URLs batch to storage", zap.Error(err))
// 		return nil, fmt.Errorf("failed to save URLs batch: %w", err)
// 	}
// 	s.logger.Info("Batch URLs saved successfully")
// 	return responseEntries, nil
// }

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
	s.logger.Debug("Ping called")
	if s.storage == nil {
		s.logger.Error("Ping: storage is nil")
		return errors.New("storage not initialized")
	}
	return s.storage.CheckConnection(ctx)
}

// generateShortURL генерирует короткий URL на основе оригинального URL
func generateShortURL(originalURL string) string {
	hash := sha256.Sum256([]byte(originalURL))
	encoded := base64.URLEncoding.EncodeToString(hash[:])
	if len(encoded) > 8 {
		return encoded[:8]
	}
	return encoded
}

// Close закрывает соединение с хранилищем (если это применимо)
func (s *URLServiceImpl) Close() error {
	s.logger.Info("Close called on URLService")
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}

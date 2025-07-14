// Package service реализует бизнес-логику для сервиса сокращения URL.
// Предоставляет сервисный слой между HTTP обработчиками и хранилищем данных.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// URLService определяет интерфейс для бизнес-логики работы с URL.
// Предоставляет методы высокого уровня для создания, получения и управления URL,
// инкапсулируя логику валидации, генерации хешей и работы с хранилищем.
type URLService interface {
	// CreateShortURL создает сокращенный URL из оригинального с валидацией и проверкой дубликатов
	CreateShortURL(ctx context.Context, originalURL string) (string, error)
	// GetOriginalURL получает оригинальный URL по короткому идентификатору
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	// GetStorage возвращает используемое хранилище (для интеграционных тестов)
	GetStorage() storage.URLStorage
	// CreateShortURLsBatch создает несколько сокращенных URL за один запрос
	CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	// CheckConnection проверяет доступность хранилища данных
	CheckConnection(ctx context.Context) error
	// GetUserURLs получает все URL пользователя с формированием полных адресов
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
	// BatchDeleteURLs выполняет массовое удаление URL с оптимизацией для больших объемов
	BatchDeleteURLs(ctx context.Context, shortURLs []string, userID string) error
}

// URLServiceImpl реализует интерфейс URLService.
// Содержит зависимости для работы с хранилищем, конфигурацией и логированием.
type URLServiceImpl struct {
	storage storage.URLStorage // Хранилище URL
	config  *config.Config     // Конфигурация приложения
	logger  *zap.Logger        // Логгер для записи событий
}

// NewURLService создает новый экземпляр URLService с автоматическим выбором хранилища.
// Приоритет выбора хранилища: PostgreSQL -> File -> Memory.
// Если PostgreSQL недоступен, автоматически переключается на файловое хранилище,
// если файловое хранилище недоступно - использует память.
//
// Параметры:
//   - cfg: конфигурация с настройками подключения к различным хранилищам
//   - logger: логгер для записи событий инициализации и работы сервиса
//
// Возвращает URLService или ошибку при критических проблемах инициализации.
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
	userID, ok := ctx.Value(middleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		// Если userID не найден в контексте, это может быть ошибкой или особенностью вызова.
		// В зависимости от требований, можно либо возвращать ошибку, либо генерировать временный userID,
		// либо использовать некий "общий" userID.
		// Для автотестов, если кука есть, userID должен быть.
		// Если куки нет (первый запрос без куки), то AuthMiddleware должен был ее создать и положить userID в контекст.
		// Таким образом, userID здесь *должен* быть, если AuthMiddleware отработал корректно.
		s.logger.Error("UserID not found in context during CreateShortURL. This should not happen if AuthMiddleware is working.")
		return "", fmt.Errorf("userID not found in context, authentication might have failed")
		// userID = "" // Предыдущая логика, которая приводила к ошибке в тесте
	}

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

	err = s.storage.Save(ctx, shortURL, originalURL, userID)
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

	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		s.logger.Error("UserID not found in context during CreateShortURLsBatch")
		return nil, fmt.Errorf("userID not found in context, authentication might have failed")
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

// GetUserURLs получает все URL, сокращенные пользователем
func (s *URLServiceImpl) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	userURLs, err := s.storage.GetUserURLs(ctx, userID)
	if err != nil {
		s.logger.Error("Error getting user URLs from storage in service", zap.String("userID", userID), zap.Error(err))
		return nil, fmt.Errorf("service: could not retrieve URLs for user %s: %w", userID, err)
	}
	// Важно: здесь shortURL из хранилища это только ID. Нужно его дополнить BaseURL.
	fullUserURLs := make([]models.UserURL, len(userURLs))
	for i, u := range userURLs {
		fullUserURLs[i] = models.UserURL{
			ShortURL:    s.config.BaseURL + "/" + u.ShortURL,
			OriginalURL: u.OriginalURL,
		}
	}
	return fullUserURLs, nil
}

// BatchDeleteURLs deletes multiple URLs using fan-in pattern
func (s *URLServiceImpl) BatchDeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	// Получаем параметры из конфигурации
	sequentialThreshold := s.config.BatchDeleteSequentialThreshold
	maxWorkers := s.config.BatchDeleteMaxWorkers
	batchSize := s.config.BatchDeleteBatchSize

	// Если URL мало, удаляем их последовательно
	if len(shortURLs) <= sequentialThreshold {
		return s.storage.BatchDelete(ctx, shortURLs, userID)
	}

	// Для большого количества URL используем паттерн fan-in

	// Создаем канал для сбора результатов от всех worker'ов (fan-in)
	errorChan := make(chan error, maxWorkers)

	// Используем WaitGroup для синхронизации завершения всех worker'ов
	var wg sync.WaitGroup

	// Разбиваем URL на батчи для параллельной обработки
	batches := make([][]string, 0)
	for i := 0; i < len(shortURLs); i += batchSize {
		end := i + batchSize
		if end > len(shortURLs) {
			end = len(shortURLs)
		}
		batches = append(batches, shortURLs[i:end])
	}

	// Ограничиваем количество worker'ов
	workers := len(batches)
	if workers > maxWorkers {
		workers = maxWorkers
	}

	s.logger.Info("Starting parallel URL deletion",
		zap.String("userID", userID),
		zap.Int("totalURLs", len(shortURLs)),
		zap.Int("batches", len(batches)),
		zap.Int("workers", workers))

	// Добавляем количество worker'ов в WaitGroup
	wg.Add(workers)

	// Запускаем worker'ы для параллельной обработки батчей
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done() // Сигнализируем о завершении worker'а

			// Каждый worker обрабатывает свою часть батчей
			for batchIndex := workerID; batchIndex < len(batches); batchIndex += workers {
				batch := batches[batchIndex]

				s.logger.Debug("Worker processing batch",
					zap.Int("workerID", workerID),
					zap.Int("batchIndex", batchIndex),
					zap.Strings("urls", batch))

				// Удаляем батч URL
				if err := s.storage.BatchDelete(ctx, batch, userID); err != nil {
					s.logger.Debug("Worker encountered error deleting batch",
						zap.Int("workerID", workerID),
						zap.Int("batchIndex", batchIndex),
						zap.Int("batchSize", len(batch)),
						zap.Error(err))

					// Отправляем ошибку в канал (fan-in)
					select {
					case errorChan <- err:
					case <-ctx.Done():
						return
					}
				} else {
					s.logger.Debug("Batch deleted successfully",
						zap.Int("workerID", workerID),
						zap.Int("batchIndex", batchIndex),
						zap.Int("count", len(batch)))
				}
			}
		}(i)
	}

	// Goroutine для закрытия errorChan после завершения всех worker'ов
	go func() {
		wg.Wait()        // Ждем завершения всех worker'ов
		close(errorChan) // Закрываем канал ошибок
	}()

	// Собираем все ошибки из канала (fan-in consumer)
	var errList []error
	for err := range errorChan {
		errList = append(errList, err)
	}

	// Если были ошибки, объединяем их в одну составную ошибку
	if len(errList) > 0 {
		// Создаем детализированное сообщение об ошибке с информацией о всех проблемах
		errorMsg := fmt.Sprintf("batch deletion failed with %d error(s): ", len(errList))
		for i, err := range errList {
			errorMsg += fmt.Sprintf("[%d] %v", i+1, err)
			if i < len(errList)-1 {
				errorMsg += "; "
			}
		}

		s.logger.Error("Batch deletion completed with errors",
			zap.String("userID", userID),
			zap.Int("totalURLs", len(shortURLs)),
			zap.Int("errorCount", len(errList)),
			zap.String("errorDetails", errorMsg))

		return errors.New(errorMsg)
	}

	s.logger.Info("Batch deletion completed successfully",
		zap.String("userID", userID),
		zap.Int("totalURLs", len(shortURLs)))

	return nil
}

package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// MemoryStorage реализует URLStorage с использованием памяти
type MemoryStorage struct {
	mu       sync.RWMutex
	urls     map[string]string
	userURLs map[string][]models.UserURL // map[userID][]UserURL
	logger   *zap.Logger
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage(logger *zap.Logger) *MemoryStorage {
	return &MemoryStorage{
		urls:     make(map[string]string),
		userURLs: make(map[string][]models.UserURL),
		logger:   logger,
	}
}

// Save сохраняет URL в памяти
func (ms *MemoryStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.urls[shortURL] = originalURL

	// Добавляем URL в список пользователя
	userURL := models.UserURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	ms.userURLs[userID] = append(ms.userURLs[userID], userURL)

	return nil
}

// Get получает оригинальный URL по короткому
func (ms *MemoryStorage) Get(ctx context.Context, shortURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if url, exists := ms.urls[shortURL]; exists {
		return url, nil
	}

	return "", ErrURLNotFound
}

// GetShortURLByOriginal получает короткий URL по оригинальному
func (ms *MemoryStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for short, orig := range ms.urls {
		if orig == originalURL {
			return short, nil
		}
	}

	return "", ErrURLNotFound
}

// SaveBatch сохраняет пакет URL
func (ms *MemoryStorage) SaveBatch(ctx context.Context, batch []BatchEntry) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, entry := range batch {
		ms.urls[entry.ShortURL] = entry.OriginalURL

		// Добавляем URL в список пользователя
		userURL := models.UserURL{
			ShortURL:    entry.ShortURL,
			OriginalURL: entry.OriginalURL,
		}
		ms.userURLs[userID] = append(ms.userURLs[userID], userURL)
	}

	return nil
}

// SaveUserURL сохраняет URL пользователя
func (ms *MemoryStorage) SaveUserURL(ctx context.Context, userID, shortURL, originalURL string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.urls[shortURL] = originalURL

	// Добавляем URL в список пользователя
	userURL := models.UserURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	ms.userURLs[userID] = append(ms.userURLs[userID], userURL)

	return nil
}

// GetUserURLs получает все URL пользователя
func (ms *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if urls, exists := ms.userURLs[userID]; exists {
		return urls, nil
	}

	return []models.UserURL{}, nil
}

// CheckConnection проверяет доступность хранилища
func (ms *MemoryStorage) CheckConnection(ctx context.Context) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.urls == nil {
		return fmt.Errorf("storage is not initialized")
	}

	return nil
}

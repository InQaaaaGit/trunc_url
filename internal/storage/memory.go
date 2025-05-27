package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// MemoryStorage реализует URLStorage с использованием памяти
type MemoryStorage struct {
	mu sync.RWMutex
	// urls map[string]string // Заменим на структуру, хранящую и UserID
	userURLs map[string]map[string]string // map[userID]map[shortURL]originalURL
	logger   *zap.Logger
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage(logger *zap.Logger) *MemoryStorage {
	return &MemoryStorage{
		// urls:   make(map[string]string),
		userURLs: make(map[string]map[string]string),
		logger:   logger,
	}
}

// Save сохраняет URL в памяти, связывая его с userID
func (ms *MemoryStorage) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, ok := ms.userURLs[userID]; !ok {
		ms.userURLs[userID] = make(map[string]string)
	}
	ms.userURLs[userID][shortURL] = originalURL
	return nil
}

// Get получает оригинальный URL по короткому
func (ms *MemoryStorage) Get(ctx context.Context, shortURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Нужно пройти по всем пользователям, чтобы найти shortURL
	// Это неэффективно, но соответствует текущей логике Get, которая не знает userID
	for _, urlsMap := range ms.userURLs {
		if originalURL, exists := urlsMap[shortURL]; exists {
			return originalURL, nil
		}
	}

	return "", ErrURLNotFound
}

// GetShortURLByOriginal получает короткий URL по оригинальному
// Эта функция также должна учитывать userID или быть глобальной.
// Пока оставим глобальной для совместимости.
func (ms *MemoryStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, urlsMap := range ms.userURLs {
		for short, orig := range urlsMap {
			if orig == originalURL {
				return short, nil
			}
		}
	}

	return "", ErrURLNotFound
}

// SaveBatch сохраняет пакет URL
// В текущей реализации SaveBatch не учитывает userID. Это нужно будет доработать.
// Пока что будем сохранять без привязки к пользователю или в "общую" область.
// Для корректной работы с пользователями, BatchEntry должен содержать UserID.
func (ms *MemoryStorage) SaveBatch(ctx context.Context, batch []BatchEntry) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Используем фиктивный userID для пакетной вставки, так как интерфейс не передает его
	const batchUserID = "__batch__"
	if _, ok := ms.userURLs[batchUserID]; !ok {
		ms.userURLs[batchUserID] = make(map[string]string)
	}

	for _, entry := range batch {
		ms.userURLs[batchUserID][entry.ShortURL] = entry.OriginalURL
	}

	return nil
}

// GetUserURLs получает все URL, сохраненные пользователем, из памяти
func (ms *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	userSpecificURLs, ok := ms.userURLs[userID]
	if !ok {
		return []models.UserURL{}, nil // Нет URL для данного пользователя
	}

	var result []models.UserURL
	for shortURL, originalURL := range userSpecificURLs {
		result = append(result, models.UserURL{
			ShortURL:    shortURL, // Нужно будет формировать полный shortURL с BaseURL
			OriginalURL: originalURL,
		})
	}

	return result, nil
}

// CheckConnection проверяет доступность хранилища
func (ms *MemoryStorage) CheckConnection(ctx context.Context) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.userURLs == nil {
		return fmt.Errorf("storage is not initialized")
	}

	return nil
}

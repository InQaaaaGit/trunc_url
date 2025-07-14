package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// URLEntry представляет запись URL в памяти
type URLEntry struct {
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

// MemoryStorage реализует URLStorage с использованием памяти
type MemoryStorage struct {
	mu sync.RWMutex
	// Изменяем структуру: map[shortURL]URLEntry
	urls   map[string]URLEntry
	logger *zap.Logger
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage(logger *zap.Logger) *MemoryStorage {
	return &MemoryStorage{
		urls:   make(map[string]URLEntry),
		logger: logger,
	}
}

// Save сохраняет URL в памяти, связывая его с userID
func (ms *MemoryStorage) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Проверка на конфликт по originalURL для данного userID
	for existingShort, entry := range ms.urls {
		if entry.OriginalURL == originalURL && entry.UserID == userID && !entry.IsDeleted && existingShort != shortURL {
			return ErrOriginalURLConflict
		}
	}

	ms.urls[shortURL] = URLEntry{
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ms *MemoryStorage) Get(ctx context.Context, shortURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entry, exists := ms.urls[shortURL]
	if !exists {
		return "", ErrURLNotFound
	}

	if entry.IsDeleted {
		return "", ErrURLDeleted
	}

	return entry.OriginalURL, nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному
// Эта функция также должна учитывать userID или быть глобальной.
// Пока оставим глобальной для совместимости.
func (ms *MemoryStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for short, entry := range ms.urls {
		if entry.OriginalURL == originalURL {
			return short, nil
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

	for _, entry := range batch {
		ms.urls[entry.ShortURL] = URLEntry{
			OriginalURL: entry.OriginalURL,
			UserID:      batchUserID,
			IsDeleted:   false,
		}
	}

	return nil
}

// GetUserURLs получает все URL, сохраненные пользователем, из памяти
func (ms *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []models.UserURL
	for shortURL, entry := range ms.urls {
		if entry.UserID == userID && !entry.IsDeleted {
			result = append(result, models.UserURL{
				ShortURL:    shortURL,
				OriginalURL: entry.OriginalURL,
			})
		}
	}

	return result, nil
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

// BatchDelete помечает URL как удаленные для указанного пользователя
func (ms *MemoryStorage) BatchDelete(ctx context.Context, shortURLs []string, userID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, shortURL := range shortURLs {
		if entry, exists := ms.urls[shortURL]; exists && entry.UserID == userID {
			entry.IsDeleted = true
			ms.urls[shortURL] = entry
		}
	}

	return nil
}

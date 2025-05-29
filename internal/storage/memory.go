package storage

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// MemoryStorage реализует URLStorage с использованием памяти
type MemoryStorage struct {
	mu     sync.RWMutex
	urls   map[string]string
	logger *zap.Logger
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage(logger *zap.Logger) *MemoryStorage {
	return &MemoryStorage{
		urls:   make(map[string]string),
		logger: logger,
	}
}

// Save сохраняет URL в памяти
func (ms *MemoryStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.urls[shortURL] = originalURL
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
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, entry := range batch {
		ms.urls[entry.ShortURL] = entry.OriginalURL
	}

	return nil
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

package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// MemoryStorage реализует хранение URL в памяти
type MemoryStorage struct {
	urls   map[string]string            // shortURL -> originalURL
	users  map[string]map[string]string // userID -> map[shortURL]originalURL
	mutex  sync.RWMutex
	logger *zap.Logger
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage(logger *zap.Logger) *MemoryStorage {
	return &MemoryStorage{
		urls:   make(map[string]string),
		users:  make(map[string]map[string]string),
		logger: logger,
	}
}

// SaveURL сохраняет пару короткий URL - оригинальный URL
func (s *MemoryStorage) SaveURL(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Сохраняем URL
	s.urls[shortURL] = originalURL

	// Сохраняем связь с пользователем
	if _, exists := s.users[userID]; !exists {
		s.users[userID] = make(map[string]string)
	}
	s.users[userID][shortURL] = originalURL

	return nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *MemoryStorage) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	originalURL, exists := s.urls[shortURL]
	if !exists {
		return "", ErrURLNotFound
	}

	return originalURL, nil
}

// GetShortURL возвращает короткий URL по оригинальному URL
func (s *MemoryStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for shortURL, url := range s.urls {
		if url == originalURL {
			return shortURL, nil
		}
	}

	return "", ErrURLNotFound
}

// GetUserURLs возвращает список URL пользователя
func (s *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userURLs, exists := s.users[userID]
	if !exists {
		return nil, nil
	}

	urls := make([]models.UserURL, 0, len(userURLs))
	for shortURL, originalURL := range userURLs {
		urls = append(urls, models.UserURL{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	return urls, nil
}

// CheckConnection проверяет соединение с хранилищем
func (s *MemoryStorage) CheckConnection(ctx context.Context) error {
	return nil // Для in-memory хранилища всегда OK
}

// Close закрывает соединение с хранилищем
func (s *MemoryStorage) Close() error {
	return nil // Для in-memory хранилища ничего не нужно закрывать
}

package storage

import "sync"

// URLStorage определяет интерфейс для хранения URL
type URLStorage interface {
	Save(shortURL, originalURL string) error
	Get(shortURL string) (string, error)
}

// MemoryStorage реализует URLStorage с использованием памяти
type MemoryStorage struct {
	urls  map[string]string
	mutex sync.RWMutex
}

// NewMemoryStorage создает новый экземпляр MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]string),
	}
}

// Save сохраняет URL в хранилище
func (s *MemoryStorage) Save(shortURL, originalURL string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.urls[shortURL] = originalURL
	return nil
}

// Get получает оригинальный URL по короткому
func (s *MemoryStorage) Get(shortURL string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if url, ok := s.urls[shortURL]; ok {
		return url, nil
	}
	return "", ErrURLNotFound
}

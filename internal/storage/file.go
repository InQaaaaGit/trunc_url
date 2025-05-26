package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// URLRecord представляет запись в файловом хранилище
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

// FileStorage реализует хранение URL в файле
type FileStorage struct {
	file   *os.File
	urls   map[string]string            // shortURL -> originalURL
	users  map[string]map[string]string // userID -> map[shortURL]originalURL
	mutex  sync.RWMutex
	logger *zap.Logger
}

// NewFileStorage создает новый экземпляр FileStorage
func NewFileStorage(filePath string, logger *zap.Logger) (*FileStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	storage := &FileStorage{
		file:   file,
		urls:   make(map[string]string),
		users:  make(map[string]map[string]string),
		logger: logger,
	}

	// Загружаем данные из файла
	if err := storage.loadFromFile(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to load data from file: %w", err)
	}

	return storage, nil
}

// SaveURL сохраняет пару короткий URL - оригинальный URL
func (s *FileStorage) SaveURL(ctx context.Context, shortURL, originalURL string) error {
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

	// Сохраняем в файл
	record := struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
		UserID      string `json:"user_id"`
	}{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *FileStorage) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	originalURL, exists := s.urls[shortURL]
	if !exists {
		return "", ErrURLNotFound
	}

	return originalURL, nil
}

// GetShortURL возвращает короткий URL по оригинальному URL
func (s *FileStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
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
func (s *FileStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
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
func (s *FileStorage) CheckConnection(ctx context.Context) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.file == nil {
		return fmt.Errorf("file is not opened")
	}

	// Проверяем, что файл доступен для записи
	if _, err := s.file.Write([]byte{}); err != nil {
		return fmt.Errorf("file is not writable: %w", err)
	}

	return nil
}

// Close закрывает соединение с хранилищем
func (s *FileStorage) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
		s.file = nil
	}

	return nil
}

// loadFromFile загружает данные из файла
func (s *FileStorage) loadFromFile() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Перемещаем указатель в начало файла
	if _, err := s.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}

	decoder := json.NewDecoder(s.file)
	for decoder.More() {
		var record struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
			UserID      string `json:"user_id"`
		}

		if err := decoder.Decode(&record); err != nil {
			return fmt.Errorf("failed to decode record: %w", err)
		}

		// Загружаем URL
		s.urls[record.ShortURL] = record.OriginalURL

		// Загружаем связь с пользователем
		if _, exists := s.users[record.UserID]; !exists {
			s.users[record.UserID] = make(map[string]string)
		}
		s.users[record.UserID][record.ShortURL] = record.OriginalURL
	}

	return nil
}

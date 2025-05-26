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

// FileStorage реализует URLStorage с использованием файла
type FileStorage struct {
	filePath string
	urls     map[string]string
	userURLs map[string][]models.UserURL // map[userID][]UserURL
	mutex    sync.RWMutex
	file     *os.File
	logger   *zap.Logger
}

// NewFileStorage создает новый экземпляр FileStorage
func NewFileStorage(filePath string, logger *zap.Logger) (*FileStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	fs := &FileStorage{
		filePath: filePath,
		file:     file,
		urls:     make(map[string]string),
		userURLs: make(map[string][]models.UserURL),
		logger:   logger,
	}

	if err := fs.loadFromFile(); err != nil {
		logger.Error("Error loading data from file", zap.Error(err))
	}

	return fs, nil
}

// loadFromFile загружает данные из файла
func (fs *FileStorage) loadFromFile() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if _, err := fs.file.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking to file start: %w", err)
	}

	decoder := json.NewDecoder(fs.file)
	for decoder.More() {
		var record URLRecord
		if err := decoder.Decode(&record); err != nil {
			return fmt.Errorf("error decoding record: %w", err)
		}
		fs.urls[record.ShortURL] = record.OriginalURL

		// Добавляем URL в список пользователя
		userURL := models.UserURL{
			ShortURL:    record.ShortURL,
			OriginalURL: record.OriginalURL,
		}
		fs.userURLs[record.UserID] = append(fs.userURLs[record.UserID], userURL)
	}

	return nil
}

// Save сохраняет URL в файл
func (fs *FileStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	record := URLRecord{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("error marshaling URL record: %w", err)
	}

	if _, err := fs.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fs.urls[shortURL] = originalURL

	// Добавляем URL в список пользователя
	userURL := models.UserURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	fs.userURLs[userID] = append(fs.userURLs[userID], userURL)

	return nil
}

// Get получает оригинальный URL по короткому
func (fs *FileStorage) Get(ctx context.Context, shortURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if url, exists := fs.urls[shortURL]; exists {
		return url, nil
	}

	return "", ErrURLNotFound
}

// SaveBatch сохраняет пакет URL
func (fs *FileStorage) SaveBatch(ctx context.Context, batch []BatchEntry) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, entry := range batch {
		record := URLRecord{
			ShortURL:    entry.ShortURL,
			OriginalURL: entry.OriginalURL,
			UserID:      userID,
		}

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("error marshaling URL record: %w", err)
		}

		if _, err := fs.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}

		fs.urls[entry.ShortURL] = entry.OriginalURL

		// Добавляем URL в список пользователя
		userURL := models.UserURL{
			ShortURL:    entry.ShortURL,
			OriginalURL: entry.OriginalURL,
		}
		fs.userURLs[userID] = append(fs.userURLs[userID], userURL)
	}

	return nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному
func (fs *FileStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	for short, orig := range fs.urls {
		if orig == originalURL {
			return short, nil
		}
	}

	return "", ErrURLNotFound
}

// SaveUserURL сохраняет URL пользователя
func (fs *FileStorage) SaveUserURL(ctx context.Context, userID, shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	record := URLRecord{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("error marshaling URL record: %w", err)
	}

	if _, err := fs.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fs.urls[shortURL] = originalURL

	// Добавляем URL в список пользователя
	userURL := models.UserURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	fs.userURLs[userID] = append(fs.userURLs[userID], userURL)

	return nil
}

// GetUserURLs получает все URL пользователя
func (fs *FileStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if urls, exists := fs.userURLs[userID]; exists {
		return urls, nil
	}

	return []models.UserURL{}, nil
}

// CheckConnection проверяет доступность файла
func (fs *FileStorage) CheckConnection(ctx context.Context) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if fs.file == nil {
		return fmt.Errorf("file is not opened")
	}

	return nil
}

// Close закрывает файл
func (fs *FileStorage) Close() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.file != nil {
		if err := fs.file.Close(); err != nil {
			return fmt.Errorf("error closing file: %w", err)
		}
		fs.file = nil
	}

	return nil
}

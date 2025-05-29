package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
)

// URLRecord represents a record in the file storage
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// FileStorage implements URLStorage using a file
type FileStorage struct {
	filePath string
	urls     map[string]string
	mutex    sync.RWMutex
	file     *os.File
	logger   *zap.Logger
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(filePath string, logger *zap.Logger) (*FileStorage, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	fs := &FileStorage{
		filePath: filePath,
		file:     file,
		urls:     make(map[string]string),
		logger:   logger,
	}

	// Load existing data from file
	if err := fs.loadFromFile(); err != nil {
		logger.Error("Error loading data from file", zap.Error(err))
		// Не возвращаем ошибку, так как файл может быть пустым
	}

	return fs, nil
}

// loadFromFile loads data from the file
func (fs *FileStorage) loadFromFile() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Перемещаем указатель в начало файла
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
	}

	return nil
}

// Save сохраняет URL в файл
func (fs *FileStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	record := URLRecord{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("error marshaling URL record: %w", err)
	}

	if _, err := fs.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fs.urls[shortURL] = originalURL
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
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, entry := range batch {
		record := URLRecord{
			ShortURL:    entry.ShortURL,
			OriginalURL: entry.OriginalURL,
		}

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("error marshaling URL record: %w", err)
		}

		if _, err := fs.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}

		fs.urls[entry.ShortURL] = entry.OriginalURL
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

// CheckConnection проверяет доступность файла
func (fs *FileStorage) CheckConnection(ctx context.Context) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if fs.file == nil {
		return fmt.Errorf("file is not open")
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

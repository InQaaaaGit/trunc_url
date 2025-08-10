package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"go.uber.org/zap"
)

// URLRecord represents a record in the file storage
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

// FileStorage implements URLStorage using a file
type FileStorage struct {
	filePath string
	urls     map[string]URLRecord
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
		urls:     make(map[string]URLRecord),
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
		fs.urls[record.ShortURL] = record
	}

	return nil
}

// Save сохраняет URL в файл, связывая его с userID
func (fs *FileStorage) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Проверка на конфликт по originalURL для данного userID
	for existingShort, record := range fs.urls {
		if record.OriginalURL == originalURL && record.UserID == userID && !record.IsDeleted && existingShort != shortURL {
			return ErrOriginalURLConflict
		}
	}

	record := URLRecord{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("error marshaling URL record: %w", err)
	}

	if _, err := fs.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fs.urls[shortURL] = record
	return nil
}

// Get получает оригинальный URL по короткому
func (fs *FileStorage) Get(ctx context.Context, shortURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if record, exists := fs.urls[shortURL]; exists {
		if record.IsDeleted {
			return "", ErrURLDeleted
		}
		return record.OriginalURL, nil
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
			UserID:      entry.UserID,
			IsDeleted:   false,
		}

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("error marshaling URL record: %w", err)
		}

		if _, err := fs.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}

		fs.urls[entry.ShortURL] = record
	}

	return nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному
func (fs *FileStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	for short, record := range fs.urls {
		if record.OriginalURL == originalURL && !record.IsDeleted {
			return short, nil
		}
	}

	return "", ErrURLNotFound
}

// GetUserURLs получает все URL, сохраненные пользователем, из файла
func (fs *FileStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	var userURLs []models.UserURL
	seenURLs := make(map[string]bool) // Для избежания дублирования

	// Необходимо переоткрыть файл для чтения с начала, так как fs.file используется для дозаписи
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY, 0644)
	if err != nil {
		fs.logger.Error("Error opening file for reading user URLs", zap.Error(err))
		return nil, fmt.Errorf("error opening file for reading: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var record URLRecord
		if err := decoder.Decode(&record); err != nil {
			// Можно логировать ошибку и продолжать, если это не критично
			fs.logger.Error("Error decoding record for user URLs", zap.Error(err))
			continue
		}
		// Проверяем уникальность и добавляем только неудаленные URL
		if record.UserID == userID && !record.IsDeleted && !seenURLs[record.ShortURL] {
			userURLs = append(userURLs, models.UserURL{
				ShortURL:    record.ShortURL,
				OriginalURL: record.OriginalURL,
			})
			seenURLs[record.ShortURL] = true
		}
	}

	if len(userURLs) == 0 {
		// Если URL-ов нет, можно вернуть пустой слайс и nil ошибку,
		// или специальную ошибку вроде ErrNoURLsFoundForUser
		return []models.UserURL{}, nil
	}

	return userURLs, nil
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

// BatchDelete помечает URL как удаленные для указанного пользователя
func (fs *FileStorage) BatchDelete(ctx context.Context, shortURLs []string, userID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Обновляем записи в памяти
	for _, shortURL := range shortURLs {
		if record, exists := fs.urls[shortURL]; exists && record.UserID == userID {
			record.IsDeleted = true
			fs.urls[shortURL] = record
		}
	}

	// Перезаписываем весь файл с обновленными данными
	// Это не самый эффективный способ, но простой для реализации
	if err := fs.rewriteFile(); err != nil {
		return fmt.Errorf("error rewriting file after batch delete: %w", err)
	}

	return nil
}

// rewriteFile перезаписывает файл с текущими данными из памяти
func (fs *FileStorage) rewriteFile() error {
	// Закрываем текущий файл
	if err := fs.file.Close(); err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	// Открываем файл для перезаписи
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file for rewrite: %w", err)
	}

	// Записываем все данные
	for _, record := range fs.urls {
		data, err := json.Marshal(record)
		if err != nil {
			file.Close()
			return fmt.Errorf("error marshaling record: %w", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			file.Close()
			return fmt.Errorf("error writing record: %w", err)
		}
	}

	// Переоткрываем файл в режиме append для дальнейшей работы
	if err := file.Close(); err != nil {
		return fmt.Errorf("error closing rewritten file: %w", err)
	}

	fs.file, err = os.OpenFile(fs.filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error reopening file: %w", err)
	}

	return nil
}

// Sync принудительно синхронизирует данные с диском
func (fs *FileStorage) Sync() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.file != nil {
		if err := fs.file.Sync(); err != nil {
			return fmt.Errorf("error syncing file: %w", err)
		}
	}

	return nil
}

// Close закрывает файл
func (fs *FileStorage) Close() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.file != nil {
		// Принудительно синхронизируем данные перед закрытием
		if err := fs.file.Sync(); err != nil {
			fs.logger.Error("Error syncing file before close", zap.Error(err))
		}

		if err := fs.file.Close(); err != nil {
			return fmt.Errorf("error closing file: %w", err)
		}
		fs.file = nil
	}

	return nil
}

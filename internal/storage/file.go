package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// URLRecord представляет запись в файле хранилища
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// FileStorage реализует URLStorage с использованием файла
type FileStorage struct {
	filePath string
	urls     map[string]string
	mutex    sync.RWMutex
}

// NewFileStorage создает новый экземпляр FileStorage
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]string),
	}

	// Загружаем существующие данные из файла
	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}

	return fs, nil
}

// loadFromFile загружает данные из файла
func (fs *FileStorage) loadFromFile() error {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var record URLRecord
		if err := decoder.Decode(&record); err != nil {
			return err
		}
		fs.urls[record.ShortURL] = record.OriginalURL
	}

	return nil
}

// saveToFile сохраняет данные в файл
func (fs *FileStorage) saveToFile() error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for shortURL, originalURL := range fs.urls {
		record := URLRecord{
			UUID:        shortURL, // Используем shortURL как UUID для простоты
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

// Save сохраняет URL в хранилище
func (fs *FileStorage) Save(shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	fs.urls[shortURL] = originalURL
	return fs.saveToFile()
}

// Get получает оригинальный URL по короткому
func (fs *FileStorage) Get(shortURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if url, ok := fs.urls[shortURL]; ok {
		return url, nil
	}
	return "", ErrURLNotFound
}

// SaveBatch сохраняет пакет URL в файловое хранилище
func (fs *FileStorage) SaveBatch(batch []BatchEntry) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Сначала загружаем текущие данные, чтобы не потерять их
	// (на случай, если были изменения с момента последнего loadFromFile)
	// В реальном приложении это может быть неэффективно для больших файлов
	if err := fs.loadFromFile(); err != nil {
		// Если не удалось прочитать, возможно, файл пуст или поврежден.
		// Попробуем продолжить, перезаписав его новыми данными.
		// Но лучше залогировать это.
		log.Printf("Предупреждение: не удалось прочитать файл %s перед пакетной записью: %v", fs.filePath, err)
		fs.urls = make(map[string]string) // Начинаем с чистого листа
	}

	// Добавляем новые записи в карту
	for _, entry := range batch {
		fs.urls[entry.ShortURL] = entry.OriginalURL
	}

	// Перезаписываем файл с обновленными данными
	return fs.saveToFile()
}

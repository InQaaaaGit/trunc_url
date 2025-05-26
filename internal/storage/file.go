package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
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
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]string),
	}

	// Load existing data from file
	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}

	return fs, nil
}

// loadFromFile loads data from the file
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

// saveToFile saves data to the file
func (fs *FileStorage) saveToFile() error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for shortURL, originalURL := range fs.urls {
		record := URLRecord{
			UUID:        shortURL, // Use shortURL as UUID for simplicity
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

// Save saves a URL to the storage
func (fs *FileStorage) Save(shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	fs.urls[shortURL] = originalURL
	return fs.saveToFile()
}

// Get retrieves the original URL from a short one
func (fs *FileStorage) Get(shortURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if url, ok := fs.urls[shortURL]; ok {
		return url, nil
	}
	return "", ErrURLNotFound
}

// SaveBatch saves a batch of URLs to the file storage
func (fs *FileStorage) SaveBatch(batch []BatchEntry) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// First, load existing data to avoid losing them
	// (in case there were changes since the last loadFromFile)
	// In a real application, this might be inefficient for large files
	if err := fs.loadFromFile(); err != nil {
		// If reading fails, it might be because the file is empty or corrupted.
		// We'll try to continue, overwriting the file with new data.
		// But it's better to log this.
		log.Printf("Warning: failed to read file %s before batch write: %v", fs.filePath, err)
		fs.urls = make(map[string]string) // Start with a clean slate
	}

	// Add new records to the map
	for _, entry := range batch {
		fs.urls[entry.ShortURL] = entry.OriginalURL
	}

	// Overwrite the file with updated data
	return fs.saveToFile()
}

// GetShortURLByOriginal finds a short URL from an original one in the file
func (fs *FileStorage) GetShortURLByOriginal(originalURL string) (string, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Reloading data before search might be inefficient,
	// but guarantees freshness if the file could have been changed from outside.
	// In the current implementation, this is unnecessary, as all changes go through this instance.
	// Simply search in the current state of fs.urls
	for short, orig := range fs.urls {
		if orig == originalURL {
			return short, nil
		}
	}
	return "", ErrURLNotFound
}

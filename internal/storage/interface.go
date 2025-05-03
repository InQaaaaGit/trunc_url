package storage

// BatchEntry используется для передачи данных при пакетном сохранении
type BatchEntry struct {
	ShortURL    string
	OriginalURL string
}

// URLStorage интерфейс для хранилища URL
type URLStorage interface {
	// Save сохраняет URL в хранилище
	Save(shortURL, originalURL string) error

	// Get получает оригинальный URL по короткому
	Get(shortURL string) (string, error)

	// GetShortURLByOriginal получает короткий URL по оригинальному
	// Возвращает ErrURLNotFound, если оригинальный URL не найден.
	GetShortURLByOriginal(originalURL string) (string, error)

	// SaveBatch сохраняет пакет URL
	SaveBatch(batch []BatchEntry) error
}

// DatabaseChecker интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection() error
}

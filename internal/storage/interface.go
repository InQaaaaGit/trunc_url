package storage

import "context"

// BatchEntry используется для передачи данных при пакетном сохранении
type BatchEntry struct {
	ShortURL    string
	OriginalURL string
}

// URLStorage интерфейс для хранилища URL
type URLStorage interface {
	// Save сохраняет URL в хранилище
	Save(ctx context.Context, shortURL, originalURL string) error

	// Get получает оригинальный URL по короткому
	Get(ctx context.Context, shortURL string) (string, error)

	// GetShortURLByOriginal получает короткий URL по оригинальному
	// Возвращает ErrURLNotFound, если оригинальный URL не найден.
	GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error)

	// SaveBatch сохраняет пакет URL
	SaveBatch(ctx context.Context, batch []BatchEntry) error
}

// DatabaseChecker интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error
}

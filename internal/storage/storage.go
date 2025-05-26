package storage

import "context"

// URLStorage определяет интерфейс для хранения URL
type URLStorage interface {
	// Save сохраняет URL в хранилище
	Save(ctx context.Context, shortURL, originalURL string) error
	// Get получает оригинальный URL по короткому
	Get(ctx context.Context, shortURL string) (string, error)
	// SaveBatch сохраняет пакет URL в хранилище
	SaveBatch(ctx context.Context, batch []BatchEntry) error
	// GetShortURLByOriginal получает короткий URL по оригинальному
	GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error)
}

// DatabaseChecker определяет интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error
}

// BatchEntry представляет запись для пакетного сохранения
type BatchEntry struct {
	ShortURL    string
	OriginalURL string
}

package storage

import (
	"context"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
)

// BatchEntry используется для передачи данных при пакетном сохранении
type BatchEntry struct {
	ShortURL    string
	OriginalURL string
	UserID      string
}

// URLStorage определяет интерфейс для хранения URL
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

	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error

	// Новые методы для работы с пользователями
	SaveUserURL(ctx context.Context, userID, shortURL, originalURL string) error
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
}

// DatabaseChecker интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error
}

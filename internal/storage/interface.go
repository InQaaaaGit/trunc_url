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
	// SaveURL сохраняет пару короткий URL - оригинальный URL
	SaveURL(ctx context.Context, shortURL, originalURL string) error

	// GetOriginalURL возвращает оригинальный URL по короткому URL
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)

	// GetShortURL возвращает короткий URL по оригинальному URL
	GetShortURL(ctx context.Context, originalURL string) (string, error)

	// GetUserURLs возвращает список URL пользователя
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)

	// CheckConnection проверяет соединение с хранилищем
	CheckConnection(ctx context.Context) error

	// Close закрывает соединение с хранилищем
	Close() error
}

// UserURL представляет URL пользователя в хранилище
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// DatabaseChecker определяет интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error
}

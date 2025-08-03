package storage

import (
	"context"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
)

// BatchEntry используется для передачи данных при пакетном сохранении
type BatchEntry struct {
	ShortURL    string
	OriginalURL string
}

// URLStorage интерфейс для хранилища URL
type URLStorage interface {
	// Save сохраняет URL в хранилище, связывая его с userID
	Save(ctx context.Context, shortURL, originalURL, userID string) error

	// Get получает оригинальный URL по короткому
	Get(ctx context.Context, shortURL string) (string, error)

	// GetShortURLByOriginal получает короткий URL по оригинальному
	// Возвращает ErrURLNotFound, если оригинальный URL не найден.
	GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error)

	// SaveBatch сохраняет пакет URL
	SaveBatch(ctx context.Context, batch []BatchEntry) error

	// GetUserURLs получает все URL, сохраненные пользователем
	// Этот метод должен быть реализован для каждого типа хранилища.
	// Он должен каким-то образом связывать URL с userID.
	// Пока что сигнатура не включает userID, так как текущая структура Save/Get его не учитывает.
	// Необходимо будет доработать Save, чтобы он принимал и сохранял userID.
	// И GetUserURLs будет использовать этот userID для выборки.
	// В текущем виде, без userID в Save, этот метод не сможет корректно работать.
	// Однако, для соответствия интерфейсу сервиса, добавим его.
	// TODO: Пересмотреть сохранение URL для включения userID.
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
}

// DatabaseChecker интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection(ctx context.Context) error
}

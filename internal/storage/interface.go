// Package storage предоставляет интерфейсы и типы для работы с различными хранилищами URL.
// Поддерживает memory, file и PostgreSQL хранилища с единым интерфейсом.
package storage

import (
	"context"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
)

// BatchEntry используется для передачи данных при пакетном сохранении URL.
// Содержит минимальную информацию, необходимую для сохранения одного URL в батче.
type BatchEntry struct {
	ShortURL    string // Короткий идентификатор URL
	OriginalURL string // Оригинальный полный URL
	UserID      string // Идентификатор пользователя, который создал URL
}

// URLStorage определяет интерфейс для хранилища URL.
// Предоставляет методы для сохранения, получения и управления URL с поддержкой
// пользователей и пакетных операций.
type URLStorage interface {
	// Save сохраняет URL в хранилище, связывая его с указанным пользователем.
	// Возвращает ошибку, если сохранение не удалось, включая ErrOriginalURLConflict
	// при попытке сохранить дублирующийся оригинальный URL.
	Save(ctx context.Context, shortURL, originalURL, userID string) error

	// Get получает оригинальный URL по короткому идентификатору.
	// Возвращает ErrURLNotFound, если URL не найден, или ErrURLDeleted,
	// если URL был помечен как удаленный.
	Get(ctx context.Context, shortURL string) (string, error)

	// GetShortURLByOriginal получает короткий URL по оригинальному.
	// Возвращает ErrURLNotFound, если оригинальный URL не найден в хранилище.
	// Используется для проверки дубликатов при создании новых URL.
	GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error)

	// SaveBatch сохраняет пакет URL за одну операцию для повышения производительности.
	// Каждый элемент BatchEntry должен содержать корректные shortURL и originalURL.
	SaveBatch(ctx context.Context, batch []BatchEntry) error

	// GetUserURLs получает все URL, сохраненные указанным пользователем.
	// Возвращает слайс UserURL с парами короткий/оригинальный URL.
	// Возвращает пустой слайс, если у пользователя нет сохраненных URL.
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)

	// BatchDelete помечает указанные URL как удаленные для конкретного пользователя.
	// Удаленные URL перестают быть доступными через Get, но остаются в хранилище.
	// Операция выполняется асинхронно и может обрабатывать большие объемы данных.
	BatchDelete(ctx context.Context, shortURLs []string, userID string) error
}

// DatabaseChecker определяет интерфейс для проверки состояния подключения к базе данных.
// Реализуется хранилищами, которые поддерживают проверку соединения (например, PostgreSQL).
type DatabaseChecker interface {
	// CheckConnection проверяет доступность соединения с базой данных.
	// Возвращает ошибку, если соединение недоступно или база данных не отвечает.
	// Используется в health check эндпоинте /ping.
	CheckConnection(ctx context.Context) error
}

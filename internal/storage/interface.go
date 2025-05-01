package storage

// URLStorage интерфейс для хранилища URL
type URLStorage interface {
	// Save сохраняет URL в хранилище и возвращает короткий URL
	Save(shortURL, originalURL string) error

	// Get получает оригинальный URL по короткому
	Get(shortURL string) (string, error)
}

// DatabaseChecker интерфейс для проверки соединения с базой данных
type DatabaseChecker interface {
	// CheckConnection проверяет соединение с базой данных
	CheckConnection() error
}

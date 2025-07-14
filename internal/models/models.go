// Package models содержит структуры данных для работы с URL и API запросами/ответами.
// Пакет определяет основные модели для передачи данных между слоями приложения.
package models

// UserURL представляет структуру для URL пользователя в API ответах.
// Используется для возврата списка сокращенных URL пользователя.
type UserURL struct {
	ShortURL    string `json:"short_url"`    // Полный сокращенный URL (с базовым адресом)
	OriginalURL string `json:"original_url"` // Оригинальный URL
}

// DeleteRequest представляет запрос на удаление URL.
// Содержит массив коротких URL для удаления.
type DeleteRequest []string

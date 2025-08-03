package models

// UserURL представляет структуру для URL пользователя
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

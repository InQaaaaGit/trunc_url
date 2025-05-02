package models

// BatchRequestEntry представляет одну запись в запросе на пакетное сокращение
type BatchRequestEntry struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponseEntry представляет одну запись в ответе на пакетное сокращение
type BatchResponseEntry struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

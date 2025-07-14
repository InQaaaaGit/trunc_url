package models

// BatchRequestEntry представляет одну запись в запросе на пакетное сокращение URL.
// Используется в API эндпоинте /api/shorten/batch для массовой обработки URL.
type BatchRequestEntry struct {
	CorrelationID string `json:"correlation_id"` // Уникальный идентификатор для связи запроса и ответа
	OriginalURL   string `json:"original_url"`   // Оригинальный URL для сокращения
}

// BatchResponseEntry представляет одну запись в ответе на пакетное сокращение.
// Возвращается в API эндпоинте /api/shorten/batch с результатами обработки.
type BatchResponseEntry struct {
	CorrelationID string `json:"correlation_id"` // Тот же идентификатор из запроса
	ShortURL      string `json:"short_url"`      // Сокращенный URL
}

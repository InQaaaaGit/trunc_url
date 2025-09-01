// Package grpc содержит gRPC обработчики для сервиса сокращения URL.
package grpc

import "context"

// Простые типы для gRPC сообщений (без proto генерации)

// CreateShortURLRequest запрос на создание короткого URL
type CreateShortURLRequest struct {
	OriginalURL string
}

// CreateShortURLResponse ответ с созданным коротким URL
type CreateShortURLResponse struct {
	ShortURL     string
	StatusCode   int32
	ErrorMessage string
}

// GetOriginalURLRequest запрос на получение оригинального URL
type GetOriginalURLRequest struct {
	ShortID string
}

// GetOriginalURLResponse ответ с оригинальным URL
type GetOriginalURLResponse struct {
	OriginalURL  string
	StatusCode   int32
	ErrorMessage string
}

// BatchRequestEntry представляет одну запись в пакетном запросе
type BatchRequestEntry struct {
	CorrelationID string
	OriginalURL   string
}

// BatchResponseEntry представляет одну запись в пакетном ответе
type BatchResponseEntry struct {
	CorrelationID string
	ShortURL      string
}

// CreateShortURLsBatchRequest запрос на пакетное создание URL
type CreateShortURLsBatchRequest struct {
	Entries []BatchRequestEntry
}

// CreateShortURLsBatchResponse ответ с пакетно созданными URL
type CreateShortURLsBatchResponse struct {
	Entries      []BatchResponseEntry
	StatusCode   int32
	ErrorMessage string
}

// UserURL представляет URL пользователя
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// GetUserURLsRequest запрос на получение URL пользователя
type GetUserURLsRequest struct {
	UserID string
}

// GetUserURLsResponse ответ с URL пользователя
type GetUserURLsResponse struct {
	URLs         []UserURL
	StatusCode   int32
	ErrorMessage string
}

// BatchDeleteURLsRequest запрос на пакетное удаление URL
type BatchDeleteURLsRequest struct {
	ShortURLs []string
	UserID    string
}

// BatchDeleteURLsResponse ответ на пакетное удаление URL
type BatchDeleteURLsResponse struct {
	StatusCode   int32
	ErrorMessage string
}

// PingRequest запрос на проверку соединения
type PingRequest struct{}

// PingResponse ответ на проверку соединения
type PingResponse struct {
	StatusCode   int32
	ErrorMessage string
}

// GetStatsRequest запрос на получение статистики
type GetStatsRequest struct{}

// GetStatsResponse ответ со статистикой
type GetStatsResponse struct {
	UrlsCount    int32
	UsersCount   int32
	StatusCode   int32
	ErrorMessage string
}

// URLServiceServer интерфейс для gRPC сервиса
type URLServiceServer interface {
	CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error)
	GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error)
	CreateShortURLsBatch(ctx context.Context, req *CreateShortURLsBatchRequest) (*CreateShortURLsBatchResponse, error)
	GetUserURLs(ctx context.Context, req *GetUserURLsRequest) (*GetUserURLsResponse, error)
	BatchDeleteURLs(ctx context.Context, req *BatchDeleteURLsRequest) (*BatchDeleteURLsResponse, error)
	Ping(ctx context.Context, req *PingRequest) (*PingResponse, error)
	GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error)
}

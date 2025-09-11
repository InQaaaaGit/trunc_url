// Package grpc содержит gRPC обработчики для сервиса сокращения URL.
// Использует OpaqueAPI подход для упрощения API и улучшения производительности.
package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ===== OpaqueAPI типы =====

// ExecuteRequest универсальный запрос для всех операций с URL
type ExecuteRequest struct {
	// OperationType определяет тип операции
	OperationType string
	// Payload содержит данные для операции в формате Any
	Payload *anypb.Any
}

// ExecuteResponse универсальный ответ для всех операций с URL
type ExecuteResponse struct {
	// Success указывает на успешность операции
	Success bool
	// StatusCode HTTP статус код
	StatusCode int32
	// ErrorMessage сообщение об ошибке (если есть)
	ErrorMessage string
	// Payload содержит результат операции в формате Any
	Payload *anypb.Any
}

// PingResponse ответ на проверку соединения
type PingResponse struct {
	Success      bool
	StatusCode   int32
	ErrorMessage string
}

// ===== Payload типы для различных операций =====

// CreateShortURLPayload данные для создания короткого URL
type CreateShortURLPayload struct {
	OriginalURL string
}

// CreateShortURLResult результат создания короткого URL
type CreateShortURLResult struct {
	ShortURL string
}

// GetOriginalURLPayload данные для получения оригинального URL
type GetOriginalURLPayload struct {
	ShortID string
}

// GetOriginalURLResult результат получения оригинального URL
type GetOriginalURLResult struct {
	OriginalURL string
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

// CreateShortURLsBatchPayload данные для пакетного создания URL
type CreateShortURLsBatchPayload struct {
	Entries []BatchRequestEntry
}

// CreateShortURLsBatchResult результат пакетного создания URL
type CreateShortURLsBatchResult struct {
	Entries []BatchResponseEntry
}

// UserURL представляет URL пользователя
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// GetUserURLsPayload данные для получения URL пользователя
type GetUserURLsPayload struct {
	UserID string
}

// GetUserURLsResult результат получения URL пользователя
type GetUserURLsResult struct {
	URLs []UserURL
}

// BatchDeleteURLsPayload данные для пакетного удаления URL
type BatchDeleteURLsPayload struct {
	ShortURLs []string
	UserID    string
}

// GetStatsResult результат получения статистики
type GetStatsResult struct {
	UrlsCount  int32
	UsersCount int32
}

// URLServiceServer интерфейс для gRPC сервиса с OpaqueAPI
type URLServiceServer interface {
	// Execute выполняет операцию с URL на основе переданного типа сообщения
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
	// Ping проверяет соединение с хранилищем
	Ping(ctx context.Context, req *emptypb.Empty) (*PingResponse, error)
}

// OperationType константы для типов операций
const (
	OperationCreateShortURL       = "create_short_url"
	OperationGetOriginalURL       = "get_original_url"
	OperationCreateShortURLsBatch = "create_short_urls_batch"
	OperationGetUserURLs          = "get_user_urls"
	OperationBatchDeleteURLs      = "batch_delete_urls"
	OperationGetStats             = "get_stats"
)

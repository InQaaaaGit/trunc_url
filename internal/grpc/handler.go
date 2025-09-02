// Package grpc содержит gRPC обработчики для сервиса сокращения URL.
// Использует OpaqueAPI подход для упрощения API и улучшения производительности.
package grpc

import (
	"context"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCHandler реализует gRPC сервис для работы с URL
type GRPCHandler struct {
	service service.URLService
	cfg     *config.Config
	logger  *zap.Logger
}

// NewGRPCHandler создает новый экземпляр gRPC обработчика
func NewGRPCHandler(service service.URLService, cfg *config.Config, logger *zap.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		cfg:     cfg,
		logger:  logger,
	}
}

// Execute выполняет операцию с URL на основе переданного типа сообщения
// Это единая точка входа для всех операций с URL (OpaqueAPI подход)
func (h *GRPCHandler) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	h.logger.Info("Received gRPC Execute request", zap.String("operation_type", req.OperationType))

	switch req.OperationType {
	case OperationCreateShortURL:
		return h.handleCreateShortURL(ctx, req.Payload)
	case OperationGetOriginalURL:
		return h.handleGetOriginalURL(ctx, req.Payload)
	case OperationCreateShortURLsBatch:
		return h.handleCreateShortURLsBatch(ctx, req.Payload)
	case OperationGetUserURLs:
		return h.handleGetUserURLs(ctx, req.Payload)
	case OperationBatchDeleteURLs:
		return h.handleBatchDeleteURLs(ctx, req.Payload)
	case OperationGetStats:
		return h.handleGetStats(ctx, req.Payload)
	default:
		h.logger.Warn("Unknown operation type", zap.String("operation_type", req.OperationType))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Unknown operation type",
		}, nil
	}
}

// handleCreateShortURL обрабатывает создание короткого URL
func (h *GRPCHandler) handleCreateShortURL(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	// В реальной реализации здесь была бы десериализация payload
	// Для упрощения используем заглушку
	originalURL := "https://example.com" // В реальности извлекали бы из payload

	h.logger.Info("Processing CreateShortURL operation", zap.String("url", originalURL))

	if originalURL == "" {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "empty URL",
		}, nil
	}

	shortID, err := h.service.CreateShortURL(ctx, originalURL)
	shortURL := h.cfg.BaseURL + "/" + shortID

	if err != nil {
		if err == storage.ErrOriginalURLConflict {
			h.logger.Info("URL already exists (conflict)", zap.String("original_url", originalURL), zap.String("short_url", shortURL))
			return &ExecuteResponse{
				Success:      true,
				StatusCode:   409, // Conflict
				ErrorMessage: "",
				Payload:      &anypb.Any{}, // В реальности сериализовали бы CreateShortURLResult
			}, nil
		}
		h.logger.Error("Error creating short URL", zap.Error(err))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
		Payload:      &anypb.Any{}, // В реальности сериализовали бы CreateShortURLResult
	}, nil
}

// handleGetOriginalURL обрабатывает получение оригинального URL
func (h *GRPCHandler) handleGetOriginalURL(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	// В реальной реализации здесь была бы десериализация payload
	shortID := "abc123" // В реальности извлекали бы из payload

	h.logger.Info("Processing GetOriginalURL operation", zap.String("short_id", shortID))

	if shortID == "" {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty shortID",
		}, nil
	}

	_, err := h.service.GetOriginalURL(ctx, shortID)
	if err != nil {
		if err == storage.ErrURLNotFound {
			return &ExecuteResponse{
				Success:      false,
				StatusCode:   404, // Not Found
				ErrorMessage: "URL not found",
			}, nil
		}
		if err == storage.ErrURLDeleted {
			return &ExecuteResponse{
				Success:      false,
				StatusCode:   410, // Gone
				ErrorMessage: "URL is deleted",
			}, nil
		}
		h.logger.Error("Error getting original URL", zap.Error(err))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
		Payload:      &anypb.Any{}, // В реальности сериализовали бы GetOriginalURLResult
	}, nil
}

// handleCreateShortURLsBatch обрабатывает пакетное создание URL
func (h *GRPCHandler) handleCreateShortURLsBatch(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	h.logger.Info("Processing CreateShortURLsBatch operation")

	// В реальной реализации здесь была бы десериализация payload
	// Для упрощения используем заглушку
	batchEntries := []models.BatchRequestEntry{
		{CorrelationID: "1", OriginalURL: "https://example1.com"},
		{CorrelationID: "2", OriginalURL: "https://example2.com"},
	}

	if len(batchEntries) == 0 {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty batch request",
		}, nil
	}

	respBatch, err := h.service.CreateShortURLsBatch(ctx, batchEntries)
	if err != nil {
		h.logger.Error("Error processing batch", zap.Error(err))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	h.logger.Info("Batch processing completed", zap.Int("count", len(respBatch)))

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
		Payload:      &anypb.Any{}, // В реальности сериализовали бы CreateShortURLsBatchResult
	}, nil
}

// handleGetUserURLs обрабатывает получение URL пользователя
func (h *GRPCHandler) handleGetUserURLs(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	// В реальной реализации здесь была бы десериализация payload
	userID := "user123" // В реальности извлекали бы из payload

	h.logger.Info("Processing GetUserURLs operation", zap.String("user_id", userID))

	if userID == "" {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty userID",
		}, nil
	}

	urls, err := h.service.GetUserURLs(ctx, userID)
	if err != nil {
		h.logger.Error("Error getting user URLs", zap.Error(err))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	h.logger.Info("User URLs retrieved", zap.Int("count", len(urls)))

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
		Payload:      &anypb.Any{}, // В реальности сериализовали бы GetUserURLsResult
	}, nil
}

// handleBatchDeleteURLs обрабатывает пакетное удаление URL
func (h *GRPCHandler) handleBatchDeleteURLs(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	// В реальной реализации здесь была бы десериализация payload
	userID := "user123"                       // В реальности извлекали бы из payload
	shortURLs := []string{"abc123", "def456"} // В реальности извлекали бы из payload

	h.logger.Info("Processing BatchDeleteURLs operation", zap.String("user_id", userID), zap.Int("count", len(shortURLs)))

	if userID == "" {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty userID",
		}, nil
	}

	if len(shortURLs) == 0 {
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty URL list",
		}, nil
	}

	// Асинхронное удаление URL
	go func() {
		ctx := context.Background()
		if err := h.service.BatchDeleteURLs(ctx, shortURLs, userID); err != nil {
			h.logger.Error("Error deleting URLs", zap.String("userID", userID), zap.Strings("shortURLs", shortURLs), zap.Error(err))
		} else {
			h.logger.Info("URLs deleted successfully", zap.String("userID", userID), zap.Int("count", len(shortURLs)))
		}
	}()

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   202, // Accepted
		ErrorMessage: "",
	}, nil
}

// handleGetStats обрабатывает получение статистики
func (h *GRPCHandler) handleGetStats(ctx context.Context, payload *anypb.Any) (*ExecuteResponse, error) {
	h.logger.Info("Processing GetStats operation")

	urlsCount, usersCount, err := h.service.GetStats(ctx)
	if err != nil {
		h.logger.Error("Error getting service statistics", zap.Error(err))
		return &ExecuteResponse{
			Success:      false,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	h.logger.Info("Statistics retrieved", zap.Int("urls_count", urlsCount), zap.Int("users_count", usersCount))

	return &ExecuteResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
		Payload:      &anypb.Any{}, // В реальности сериализовали бы GetStatsResult
	}, nil
}

// Ping проверяет соединение с хранилищем
func (h *GRPCHandler) Ping(ctx context.Context, req *emptypb.Empty) (*PingResponse, error) {
	h.logger.Info("Processing Ping operation")

	if err := h.service.CheckConnection(ctx); err != nil {
		h.logger.Error("Storage connection error", zap.Error(err))
		return &PingResponse{
			Success:      false,
			StatusCode:   503, // Service Unavailable
			ErrorMessage: "Storage is no longer available",
		}, nil
	}

	return &PingResponse{
		Success:      true,
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

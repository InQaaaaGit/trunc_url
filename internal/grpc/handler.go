// Package grpc содержит gRPC обработчики для сервиса сокращения URL.
// Этот пакет предоставляет gRPC API для создания, получения и управления сокращенными URL.
package grpc

import (
	"context"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
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

// CreateShortURL создает короткий URL из оригинального
func (h *GRPCHandler) CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error) {
	originalURL := strings.TrimSpace(req.OriginalURL)
	h.logger.Info("Received URL in gRPC CreateShortURL request", zap.String("url", originalURL))

	if originalURL == "" {
		return &CreateShortURLResponse{
			ShortURL:     "",
			StatusCode:   400, // Bad Request
			ErrorMessage: "empty URL",
		}, nil
	}

	shortID, err := h.service.CreateShortURL(ctx, originalURL)
	shortURL := h.cfg.BaseURL + "/" + shortID

	if err != nil {
		if err == storage.ErrOriginalURLConflict {
			h.logger.Info("URL already exists (conflict) in gRPC", zap.String("original_url", originalURL), zap.String("short_url", shortURL))
			return &CreateShortURLResponse{
				ShortURL:     shortURL,
				StatusCode:   409, // Conflict
				ErrorMessage: "",
			}, nil
		}
		h.logger.Error("Error creating short URL in gRPC", zap.Error(err))
		return &CreateShortURLResponse{
			ShortURL:     "",
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	return &CreateShortURLResponse{
		ShortURL:     shortURL,
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

// GetOriginalURL получает оригинальный URL по короткому идентификатору
func (h *GRPCHandler) GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	shortID := strings.TrimSpace(req.ShortID)
	if shortID == "" {
		return &GetOriginalURLResponse{
			OriginalURL:  "",
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty shortID",
		}, nil
	}

	h.logger.Info("Attempting to get original URL via gRPC", zap.String("short_id", shortID))

	originalURL, err := h.service.GetOriginalURL(ctx, shortID)
	if err != nil {
		if err == storage.ErrURLNotFound {
			return &GetOriginalURLResponse{
				OriginalURL:  "",
				StatusCode:   404, // Not Found
				ErrorMessage: "URL not found",
			}, nil
		}
		if err == storage.ErrURLDeleted {
			return &GetOriginalURLResponse{
				OriginalURL:  "",
				StatusCode:   410, // Gone
				ErrorMessage: "URL is deleted",
			}, nil
		}
		h.logger.Error("Error getting original URL via gRPC", zap.Error(err))
		return &GetOriginalURLResponse{
			OriginalURL:  "",
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	return &GetOriginalURLResponse{
		OriginalURL:  originalURL,
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

// CreateShortURLsBatch создает несколько коротких URL за один запрос
func (h *GRPCHandler) CreateShortURLsBatch(ctx context.Context, req *CreateShortURLsBatchRequest) (*CreateShortURLsBatchResponse, error) {
	if len(req.Entries) == 0 {
		return &CreateShortURLsBatchResponse{
			Entries:      []BatchResponseEntry{},
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty batch request",
		}, nil
	}

	// Конвертируем записи в модели
	batchEntries := make([]models.BatchRequestEntry, len(req.Entries))
	for i, entry := range req.Entries {
		batchEntries[i] = models.BatchRequestEntry{
			CorrelationID: entry.CorrelationID,
			OriginalURL:   entry.OriginalURL,
		}
	}

	respBatch, err := h.service.CreateShortURLsBatch(ctx, batchEntries)
	if err != nil {
		h.logger.Error("Error processing batch via gRPC", zap.Error(err))
		return &CreateShortURLsBatchResponse{
			Entries:      []BatchResponseEntry{},
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	// Конвертируем ответ обратно
	protoEntries := make([]BatchResponseEntry, len(respBatch))
	for i, entry := range respBatch {
		protoEntries[i] = BatchResponseEntry{
			CorrelationID: entry.CorrelationID,
			ShortURL:      entry.ShortURL,
		}
	}

	return &CreateShortURLsBatchResponse{
		Entries:      protoEntries,
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

// GetUserURLs получает все URL пользователя
func (h *GRPCHandler) GetUserURLs(ctx context.Context, req *GetUserURLsRequest) (*GetUserURLsResponse, error) {
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return &GetUserURLsResponse{
			URLs:         []UserURL{},
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty userID",
		}, nil
	}

	urls, err := h.service.GetUserURLs(ctx, userID)
	if err != nil {
		h.logger.Error("Error getting user URLs via gRPC", zap.Error(err))
		return &GetUserURLsResponse{
			URLs:         []UserURL{},
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	// Конвертируем в наши типы
	protoURLs := make([]UserURL, len(urls))
	for i, url := range urls {
		protoURLs[i] = UserURL{
			ShortURL:    url.ShortURL,
			OriginalURL: url.OriginalURL,
		}
	}

	return &GetUserURLsResponse{
		URLs:         protoURLs,
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

// BatchDeleteURLs помечает URL как удаленные
func (h *GRPCHandler) BatchDeleteURLs(ctx context.Context, req *BatchDeleteURLsRequest) (*BatchDeleteURLsResponse, error) {
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return &BatchDeleteURLsResponse{
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty userID",
		}, nil
	}

	if len(req.ShortURLs) == 0 {
		return &BatchDeleteURLsResponse{
			StatusCode:   400, // Bad Request
			ErrorMessage: "Empty URL list",
		}, nil
	}

	// Асинхронное удаление URL
	go func() {
		ctx := context.Background()
		if err := h.service.BatchDeleteURLs(ctx, req.ShortURLs, userID); err != nil {
			h.logger.Error("Error deleting URLs via gRPC",
				zap.String("userID", userID),
				zap.Strings("shortURLs", req.ShortURLs),
				zap.Error(err))
		} else {
			h.logger.Info("URLs deleted successfully via gRPC",
				zap.String("userID", userID),
				zap.Int("count", len(req.ShortURLs)))
		}
	}()

	return &BatchDeleteURLsResponse{
		StatusCode:   202, // Accepted
		ErrorMessage: "",
	}, nil
}

// Ping проверяет соединение с хранилищем
func (h *GRPCHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	if err := h.service.CheckConnection(ctx); err != nil {
		h.logger.Error("Ошибка подключения к хранилищу via gRPC", zap.Error(err))
		return &PingResponse{
			StatusCode:   503, // Service Unavailable
			ErrorMessage: "Storage is no longer available",
		}, nil
	}

	return &PingResponse{
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

// GetStats возвращает статистику сервиса
func (h *GRPCHandler) GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error) {
	urlsCount, usersCount, err := h.service.GetStats(ctx)
	if err != nil {
		h.logger.Error("Error getting service statistics via gRPC", zap.Error(err))
		return &GetStatsResponse{
			UrlsCount:    0,
			UsersCount:   0,
			StatusCode:   500, // Internal Server Error
			ErrorMessage: "Internal server error",
		}, nil
	}

	return &GetStatsResponse{
		UrlsCount:    int32(urlsCount),
		UsersCount:   int32(usersCount),
		StatusCode:   200, // OK
		ErrorMessage: "",
	}, nil
}

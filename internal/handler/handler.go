package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const (
	contentTypePlain   = "text/plain"
	contentTypeJSON    = "application/json"
	emptyURLMessage    = "empty URL"
	invalidURLMessage  = "Invalid URL"
	urlNotFoundMessage = "URL not found"
)

// URLService определяет интерфейс для работы с URL
type URLService interface {
	CreateShortURL(ctx context.Context, url string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetStorage() storage.URLStorage
	CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	CheckConnection(ctx context.Context) error
	GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error)
}

type Handler struct {
	service service.URLService
	cfg     *config.Config
	logger  *zap.Logger
}

func NewHandler(service service.URLService, cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
		logger:  logger,
	}
}

// HandleCreateURL обрабатывает POST запросы для создания коротких URL
func (h *Handler) HandleCreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypePlain) && !strings.Contains(contentType, "gzip") && !strings.Contains(contentType, "application/x-gzip") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	shortURL, err := h.service.CreateShortURL(r.Context(), originalURL)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrInvalidURL):
			h.logger.Info("invalid URL format", zap.String("url", originalURL))
			http.Error(w, "invalid URL format", http.StatusBadRequest)
		case errors.Is(err, storage.ErrOriginalURLConflict):
			h.logger.Info("URL already exists", zap.String("url", originalURL))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			if _, err := w.Write([]byte(shortURL)); err != nil {
				h.logger.Error("failed to write response", zap.Error(err))
				return
			}
		default:
			h.logger.Error("failed to create short URL", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		return
	}
}

// HandleRedirect обрабатывает GET запросы для перенаправления на оригинальный URL
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := chi.URLParam(r, "id")
	if shortID == "" {
		http.Error(w, "invalid short URL ID", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrURLNotFound):
			h.logger.Info("URL not found", zap.String("shortID", shortID))
			http.Error(w, "URL not found", http.StatusBadRequest)
		case errors.Is(err, storage.ErrInvalidURL):
			h.logger.Info("invalid URL format", zap.String("shortID", shortID))
			http.Error(w, "invalid URL format", http.StatusBadRequest)
		default:
			h.logger.Error("failed to get original URL", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

// HandleShortenURL обрабатывает POST запросы для создания коротких URL через API
func (h *Handler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "empty URL in request", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrInvalidURL):
			h.logger.Info("invalid URL format", zap.String("url", req.URL))
			http.Error(w, "invalid URL format", http.StatusBadRequest)
		case errors.Is(err, storage.ErrOriginalURLConflict):
			h.logger.Info("URL already exists", zap.String("url", req.URL))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(map[string]string{"result": shortURL}); err != nil {
				h.logger.Error("failed to encode response", zap.Error(err))
				return
			}
		default:
			h.logger.Error("failed to create short URL", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"result": shortURL}); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		return
	}
}

// HandleShortenBatch обрабатывает POST запросы для пакетного создания коротких URL
func (h *Handler) HandleShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var req []struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return
	}

	batch := make([]service.URLBatchItem, len(req))
	for i, item := range req {
		if item.OriginalURL == "" {
			http.Error(w, "empty URL in request", http.StatusBadRequest)
			return
		}
		batch[i] = service.URLBatchItem{
			CorrelationID: item.CorrelationID,
			OriginalURL:   item.OriginalURL,
		}
	}

	results, err := h.service.CreateShortURLsBatch(r.Context(), batch)
	if err != nil {
		h.logger.Error("failed to create short URLs batch", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]string, len(results))
	for i, result := range results {
		response[i] = map[string]string{
			"correlation_id": result.CorrelationID,
			"short_url":      result.ShortURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		return
	}
}

// HandlePing обрабатывает запрос на проверку соединения с базой данных
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	if err := h.service.Ping(ctx); err != nil {
		h.logger.Error("Ошибка подключения к хранилищу", zap.Error(err))
		http.Error(w, "Storage is no longer available", http.StatusGone)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetUserURLs обрабатывает GET запросы для получения списка URL пользователя
func (h *Handler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := r.Context().Value(middleware.UserIDKey).(string); !ok {
		h.logger.Error("user ID not found in context")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetUserURLs(r.Context())
	if err != nil {
		h.logger.Error("failed to get user URLs", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]map[string]string, len(urls))
	for i, url := range urls {
		response[i] = map[string]string{
			"short_url":    url.ShortURL,
			"original_url": url.OriginalURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		return
	}
}

// WithLogging добавляет логирование запросов
func (h *Handler) WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		latency := time.Since(start)
		h.logger.Info("Request processed",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.Duration("latency", latency),
			zap.Int("status", ww.Status()),
			zap.Int("size", ww.BytesWritten()),
		)
	})
}

// WithGzip добавляет поддержку gzip сжатия
func (h *Handler) WithGzip(next http.Handler) http.Handler {
	return middleware.GzipMiddleware(next)
}

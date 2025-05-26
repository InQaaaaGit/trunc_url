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

// HandleCreateURL обрабатывает POST запрос для создания короткого URL
func (h *Handler) HandleCreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypePlain) && !strings.HasPrefix(contentType, "application/x-gzip") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Error("Error closing request body", zap.Error(err))
		}
	}()

	originalURL := strings.TrimSpace(string(body))
	h.logger.Info("Received URL in POST request", zap.String("url", originalURL))

	if originalURL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.service.CreateShortURL(r.Context(), originalURL)
	shortURL := h.cfg.BaseURL + "/" + shortID

	if err != nil {
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			h.logger.Info("URL already exists (conflict)", zap.String("original_url", originalURL), zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", contentTypePlain)
			w.WriteHeader(http.StatusConflict)
			if _, writeErr := w.Write([]byte(shortURL)); writeErr != nil {
				h.logger.Error("Error writing response (conflict)", zap.Error(writeErr))
			}
			return
		}
		h.logger.Error("Error creating short URL", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypePlain)
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		h.logger.Error("Error writing response", zap.Error(err))
	}
}

// HandleRedirect обрабатывает GET запрос для перенаправления по короткому URL
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := strings.Trim(r.URL.Path, "/")
	if shortID == "" {
		http.Error(w, "Empty shortID", http.StatusBadRequest)
		return
	}

	h.logger.Info("Attempting to get original URL", zap.String("short_id", shortID))

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortID)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			http.Error(w, urlNotFoundMessage, http.StatusBadRequest)
			return
		}
		h.logger.Error("Error getting original URL", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Setting Location header", zap.String("location", originalURL))
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

// HandleShortenURL обрабатывает POST запрос для создания короткого URL в формате JSON
func (h *Handler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypeJSON) {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Error("Error closing request body", zap.Error(err))
		}
	}()

	if req.URL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.service.CreateShortURL(r.Context(), req.URL)
	shortURL := h.cfg.BaseURL + "/" + shortID
	response := ShortenResponse{
		Result: shortURL,
	}

	if err != nil {
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			h.logger.Info("URL already exists (conflict) in /api/shorten", zap.String("original_url", req.URL), zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Error("Error writing JSON response (conflict)", zap.Error(err))
			}
			return
		}

		h.logger.Error("Error creating short URL in /api/shorten", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Error writing JSON response", zap.Error(err))
	}
}

// HandleShortenBatch обрабатывает POST запрос для пакетного создания коротких URL
func (h *Handler) HandleShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypeJSON) {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var reqBatch []models.BatchRequestEntry
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Error("Error closing request body", zap.Error(err))
		}
	}()

	if err := json.Unmarshal(bodyBytes, &reqBatch); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	respBatch, err := h.service.CreateShortURLsBatch(r.Context(), reqBatch)
	if err != nil {
		h.logger.Error("Error processing batch", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(respBatch); err != nil {
		h.logger.Error("Error writing response", zap.Error(err))
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

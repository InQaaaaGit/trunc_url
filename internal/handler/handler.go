package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	CreateShortURL(ctx context.Context, url string, userID string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetStorage() storage.URLStorage
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

// validateURL проверяет корректность URL
func validateURL(urlStr string) error {
	if urlStr == "" {
		return errors.New("empty URL")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("URL must have scheme and host")
	}

	return nil
}

// HandleCreateURL обрабатывает POST запросы для создания коротких URL
func (h *Handler) HandleCreateURL(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("HandleCreateURL called", zap.String("method", r.Method), zap.String("path", r.URL.Path))
	if r.Method != http.MethodPost {
		h.logger.Warn("HandleCreateURL: method not POST", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("HandleCreateURL: failed to read request body", zap.Error(err))
		http.Error(w, "Cannot read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		h.logger.Warn("HandleCreateURL: request body is empty")
		http.Error(w, "Request body must not be empty", http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	userID := "anonymous" // Для iteration1_test.go userID не важен, но для других итераций он может приходить из middleware
	// Попытка получить userID из контекста, если он там есть (например, установлен middleware.WithAuth)
	if idFromCtx, ok := r.Context().Value(middleware.UserIDKey).(string); ok && idFromCtx != "" {
		userID = idFromCtx
		h.logger.Debug("HandleCreateURL: UserID from context", zap.String("userID", userID))
	}

	h.logger.Info("HandleCreateURL: attempting to create short URL", zap.String("originalURL", originalURL), zap.String("userID", userID))
	shortID, err := h.service.CreateShortURL(r.Context(), originalURL, userID)
	if err != nil {
		// Проверяем, была ли ошибка связана с тем, что URL уже существует
		// В service.CreateShortURL уже есть логика, которая возвращает (shortID, nil) если storage.ErrURLAlreadyExists
		// Поэтому здесь мы должны получать ошибку только если это НЕ ErrURLAlreadyExists, либо если сам сервис решил вернуть другую ошибку.
		h.logger.Error("HandleCreateURL: service.CreateShortURL failed", zap.Error(err), zap.String("originalURL", originalURL))

		// Если ошибка storage.ErrURLAlreadyExists была проброшена до сюда (хотя не должна по текущей логике сервиса)
		// То нужно вернуть 201, а не 500. Но для этого shortID должен быть не пустым.
		// Однако, тест iteration1 ожидает 500, если POST возвращает ошибку, отличную от "URL уже существует"
		// и тест iteration8 ожидает 409 Conflict если оригинальный URL уже существует.
		// Текущая логика сервиса CreateShortURL возвращает (shortID, nil) при ErrURLAlreadyExists, что приводит к 201.
		// Если бы сервис возвращал (shortID, ErrURLAlreadyExists), то здесь нужно было бы:
		/*
		   if errors.Is(err, storage.ErrURLAlreadyExists) && shortID != "" {
		       w.Header().Set("Content-Type", contentTypePlain)
		       w.WriteHeader(http.StatusConflict) // 409 Conflict
		       fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseURL, shortID)
		       fmt.Fprint(w, fullURL)
		       return
		   }
		*/
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError) // Общая ошибка для теста iteration1
		return
	}

	fullURL := fmt.Sprintf("%s/%s", h.cfg.BaseURL, shortID)
	w.Header().Set("Content-Type", contentTypePlain) // iteration1_test ожидает text/plain
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, fullURL)
	h.logger.Info("HandleCreateURL: successfully created short URL", zap.String("originalURL", originalURL), zap.String("shortURL", fullURL), zap.String("userID", userID))
}

// HandleRedirect обрабатывает GET запросы для перенаправления на оригинальный URL
func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("HandleRedirect called", zap.String("method", r.Method), zap.String("path", r.URL.Path))
	if r.Method != http.MethodGet {
		h.logger.Warn("HandleRedirect: method not GET", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := chi.URLParam(r, "id")
	if shortID == "" {
		h.logger.Warn("HandleRedirect: shortID is empty")
		http.Error(w, "Invalid short URL ID", http.StatusBadRequest)
		return
	}

	h.logger.Info("HandleRedirect: attempting to get original URL for shortID", zap.String("shortID", shortID))
	originalURL, err := h.service.GetOriginalURL(r.Context(), shortID)
	if err != nil {
		h.logger.Warn("HandleRedirect: original URL not found or service error", zap.String("shortID", shortID), zap.Error(err))
		if errors.Is(err, storage.ErrURLNotFound) {
			http.Error(w, urlNotFoundMessage, http.StatusNotFound) // 404 Not Found, как может ожидать тест
		} else if errors.Is(err, storage.ErrInvalidURL) { // Если сервис вернул ErrInvalidURL для пустого/невалидного shortID
			http.Error(w, "Invalid short URL ID", http.StatusBadRequest)
		} else {
			http.Error(w, "Failed to retrieve original URL", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Info("HandleRedirect: redirecting", zap.String("shortID", shortID), zap.String("originalURL", originalURL))
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect) // 307 Temporary Redirect
}

// HandleShortenURL обрабатывает POST запросы для создания коротких URL через API
func (h *Handler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("HandleShortenURL called")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, contentTypeJSON) {
		h.logger.Warn("HandleShortenURL: invalid content type", zap.String("contentType", contentType))
		http.Error(w, "Invalid content type. Expected application/json", http.StatusBadRequest)
		return
	}

	var req models.ShortenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			h.logger.Warn("HandleShortenURL: empty request body")
			http.Error(w, "Empty request body", http.StatusBadRequest)
		} else {
			h.logger.Error("HandleShortenURL: failed to decode request JSON", zap.Error(err))
			http.Error(w, "Invalid request format", http.StatusBadRequest)
		}
		return
	}
	defer r.Body.Close()

	if err := validateURL(req.URL); err != nil {
		h.logger.Warn("HandleShortenURL: invalid URL in JSON request", zap.String("url", req.URL), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := "anonymous"
	if idFromCtx, ok := r.Context().Value(middleware.UserIDKey).(string); ok && idFromCtx != "" {
		userID = idFromCtx
	}
	h.logger.Info("HandleShortenURL: creating short URL", zap.String("originalURL", req.URL), zap.String("userID", userID))

	shortID, err := h.service.CreateShortURL(r.Context(), req.URL, userID)

	baseURL := strings.TrimRight(h.cfg.BaseURL, "/")
	fullShortURL := fmt.Sprintf("%s/%s", baseURL, shortID)

	if err != nil {
		// Логика обработки ошибки для /api/shorten, которая ожидает 409 при конфликте (iteration8)
		if errors.Is(err, storage.ErrURLAlreadyExists) {
			h.logger.Info("HandleShortenURL: URL already exists, returning existing one with 409 Conflict", zap.String("originalURL", req.URL), zap.String("shortURL", fullShortURL))
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusConflict) // 409 Conflict
			json.NewEncoder(w).Encode(models.ShortenResponse{Result: fullShortURL})
			return
		}
		h.logger.Error("HandleShortenURL: service.CreateShortURL failed", zap.Error(err), zap.String("originalURL", req.URL))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := models.ShortenResponse{Result: fullShortURL}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if errEnc := json.NewEncoder(w).Encode(resp); errEnc != nil {
		h.logger.Error("HandleShortenURL: failed to encode response JSON", zap.Error(errEnc))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandlePing обрабатывает запрос на проверку соединения с базой данных
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("HandlePing called")
	if err := h.service.Ping(r.Context()); err != nil {
		h.logger.Error("HandlePing: service ping failed", zap.Error(err))
		http.Error(w, "Database ping failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleGetUserURLs обрабатывает GET запросы для получения списка URL пользователя
func (h *Handler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("HandleGetUserURLs called")
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.logger.Warn("HandleGetUserURLs: UserID not found in context or empty")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userURLs, err := h.service.GetUserURLs(r.Context(), userID)
	if err != nil {
		h.logger.Error("HandleGetUserURLs: service.GetUserURLs failed", zap.String("userID", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(userURLs) == 0 {
		h.logger.Info("HandleGetUserURLs: no URLs found for user", zap.String("userID", userID))
		w.WriteHeader(http.StatusNoContent) // 204 No Content, если URL-ов нет
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(userURLs); err != nil {
		h.logger.Error("HandleGetUserURLs: failed to encode response JSON", zap.String("userID", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// WithLogging добавляет логирование запросов
func (h *Handler) WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			h.logger.Info("Request completed",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", wrapper.Status()),
				zap.Int("size", wrapper.BytesWritten()),
			)
		}()
		next.ServeHTTP(wrapper, r)
	})
}

// WithGzip добавляет поддержку gzip сжатия
func (h *Handler) WithGzip(next http.Handler) http.Handler {
	return middleware.GzipMiddleware(next)
}

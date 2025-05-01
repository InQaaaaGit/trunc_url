package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	contentTypePlain   = "text/plain"
	contentTypeJSON    = "application/json"
	emptyURLMessage    = "Empty URL"
	invalidURLMessage  = "Invalid URL"
	urlNotFoundMessage = "URL not found"
)

// URLService определяет интерфейс для работы с URL
type URLService interface {
	CreateShortURL(url string) (string, error)
	GetOriginalURL(shortID string) (string, error)
	GetStorage() interface{}
}

type Handler struct {
	service service.URLService
	cfg     *config.Config
	logger  *zap.Logger
}

func NewHandler(service service.URLService, cfg *config.Config) *Handler {
	logger, _ := zap.NewProduction()
	return &Handler{
		service: service,
		cfg:     cfg,
		logger:  logger,
	}
}

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
			h.logger.Error("Ошибка при закрытии тела запроса", zap.Error(err))
		}
	}()

	originalURL := strings.TrimSpace(string(body))
	h.logger.Info("Получен URL в POST запросе", zap.String("url", originalURL))

	if originalURL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.service.CreateShortURL(originalURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := h.cfg.BaseURL + "/" + shortID
	h.logger.Info("Создана короткая ссылка", zap.String("url", shortURL))

	w.Header().Set("Content-Type", contentTypePlain)
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		h.logger.Error("Ошибка при записи ответа", zap.Error(err))
		return
	}
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Получен запрос", zap.String("метод", r.Method), zap.String("путь", r.URL.Path))

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем shortID из пути URL или из параметра
	shortID := chi.URLParam(r, "shortID")
	if shortID == "" {
		shortID = strings.TrimPrefix(r.URL.Path, "/")
	}
	h.logger.Info("Извлечен shortID", zap.String("shortID", shortID))

	if shortID == "" {
		h.logger.Warn("Пустой shortID")
		http.Error(w, invalidURLMessage, http.StatusBadRequest)
		return
	}

	h.logger.Info("Попытка получить оригинальный URL", zap.String("shortID", shortID))
	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		h.logger.Warn("URL не найден", zap.String("shortID", shortID), zap.Error(err))
		http.Error(w, urlNotFoundMessage, http.StatusNotFound)
		return
	}

	h.logger.Info("Установка заголовка Location", zap.String("url", originalURL))
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

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
			h.logger.Error("Ошибка при закрытии тела запроса", zap.Error(err))
		}
	}()

	if req.URL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.service.CreateShortURL(req.URL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := h.cfg.BaseURL + "/" + shortID
	h.logger.Info("Создана короткая ссылка", zap.String("url", shortURL))

	response := ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Ошибка при записи ответа", zap.Error(err))
		return
	}
}

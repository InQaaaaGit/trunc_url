package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
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
	shortURL := h.cfg.BaseURL + "/" + shortID

	if err != nil {
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			h.logger.Info("URL уже существует (конфликт)", zap.String("original_url", originalURL), zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", contentTypePlain)
			w.WriteHeader(http.StatusConflict)
			if _, writeErr := w.Write([]byte(shortURL)); writeErr != nil {
				h.logger.Error("Ошибка при записи ответа (конфликт)", zap.Error(writeErr))
			}
			return
		}

		h.logger.Error("Ошибка сервиса при создании URL", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

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
	shortURL := h.cfg.BaseURL + "/" + shortID
	response := ShortenResponse{
		Result: shortURL,
	}

	if err != nil {
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			h.logger.Info("URL уже существует (конфликт) в /api/shorten", zap.String("original_url", req.URL), zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Error("Ошибка при записи JSON ответа (конфликт)", zap.Error(err))
			}
			return
		}

		h.logger.Error("Ошибка сервиса при создании URL в /api/shorten", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Создана короткая ссылка", zap.String("url", shortURL))
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Ошибка при записи JSON ответа", zap.Error(err))
		return
	}
}

// HandleShortenBatch обрабатывает запросы на пакетное сокращение URL
func (h *Handler) HandleShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверка Content-Type (должен быть application/json)
	// Учитываем возможное сжатие gzip, которое обрабатывается middleware
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypeJSON) {
		// Если Content-Type не application/json, но тело было успешно разжато GzipMiddleware,
		// возможно, исходный Content-Type был application/json
		// GzipMiddleware должен был сохранить исходный Content-Type в какой-то заголовок или контекст?
		// Проверим стандартный Accept-Encoding - не то.
		// Проще всего положиться на то, что если пришел не json, то Decode вернет ошибку.
		// Просто залогируем предупреждение, если Content-Type не json
		if !strings.Contains(contentType, "application/json") {
			h.logger.Warn("Request Content-Type is not application/json", zap.String("content-type", contentType))
		}
	}

	var reqBatch []models.BatchRequestEntry // Используем модель из пакета models
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(bodyBytes, &reqBatch); err != nil {
		h.logger.Error("Error decoding batch request JSON", zap.Error(err), zap.ByteString("body", bodyBytes))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Проверка на пустой батч
	if len(reqBatch) == 0 {
		h.logger.Info("Received empty batch request")
		http.Error(w, "Empty batch is not allowed", http.StatusBadRequest)
		return
	}

	// Вызываем метод сервиса для пакетной обработки
	respBatch, err := h.service.CreateShortURLsBatch(reqBatch) // Передаем []models.BatchRequestEntry
	if err != nil {
		h.logger.Error("Error processing batch in service", zap.Error(err))
		// Определяем, какую ошибку вернуть клиенту
		// TODO: Возможно, стоит возвращать 400 Bad Request, если ошибка связана с невалидными данными в батче?
		// Пока возвращаем 500
		http.Error(w, "Internal server error during batch processing", http.StatusInternalServerError)
		return
	}

	// Если сервис вернул пустой слайс (например, все URL были невалидны)
	if len(respBatch) == 0 {
		// Спорный момент: возвращать пустой массив с кодом 201 или ошибку 400?
		// Вернем 400, т.к. по факту ничего не было создано.
		http.Error(w, "All URLs in the batch were invalid or empty", http.StatusBadRequest)
		return
	}

	// Кодируем ответ
	respBody, err := json.Marshal(respBatch)
	if err != nil {
		h.logger.Error("Error encoding batch response JSON", zap.Error(err))
		http.Error(w, "Internal server error during response encoding", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated) // Успешное создание
	if _, err := w.Write(respBody); err != nil {
		h.logger.Error("Error writing batch response", zap.Error(err))
	}
}

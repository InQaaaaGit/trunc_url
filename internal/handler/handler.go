package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
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
}

type Handler struct {
	urlService service.URLService
	cfg        *config.Config
}

func NewHandler(urlService service.URLService, cfg *config.Config) *Handler {
	return &Handler{
		urlService: urlService,
		cfg:        cfg,
	}
}

func (h *Handler) HandleCreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypePlain) {
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
			log.Printf("Ошибка при закрытии тела запроса: %v", err)
		}
	}()

	originalURL := strings.TrimSpace(string(body))
	log.Printf("Получен URL в POST запросе: %s", originalURL)

	if originalURL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.urlService.CreateShortURL(originalURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := h.cfg.BaseURL + "/" + shortID
	log.Printf("Создана короткая ссылка: %s", shortURL)

	w.Header().Set("Content-Type", contentTypePlain)
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		log.Printf("Ошибка при записи ответа: %v", err)
		return
	}
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	log.Printf("Получен запрос: метод=%s, путь=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем shortID из пути URL или из параметра
	shortID := chi.URLParam(r, "shortID")
	if shortID == "" {
		shortID = strings.TrimPrefix(r.URL.Path, "/")
	}
	log.Printf("Извлечен shortID: %s", shortID)

	if shortID == "" {
		log.Printf("Пустой shortID")
		http.Error(w, invalidURLMessage, http.StatusBadRequest)
		return
	}

	log.Printf("Попытка получить оригинальный URL для shortID: %s", shortID)
	originalURL, err := h.urlService.GetOriginalURL(shortID)
	if err != nil {
		log.Printf("URL не найден для shortID: %s", shortID)
		http.Error(w, urlNotFoundMessage, http.StatusNotFound)
		return
	}

	log.Printf("Установка заголовка Location: %s", originalURL)
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
			log.Printf("Ошибка при закрытии тела запроса: %v", err)
		}
	}()

	if req.URL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.urlService.CreateShortURL(req.URL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := h.cfg.BaseURL + "/" + shortID
	log.Printf("Создана короткая ссылка: %s", shortURL)

	response := ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Ошибка при записи ответа: %v", err)
		return
	}
}

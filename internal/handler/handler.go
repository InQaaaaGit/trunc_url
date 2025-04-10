package handler

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	urlService service.URLService
}

func NewHandler(urlService service.URLService) *Handler {
	return &Handler{
		urlService: urlService,
	}
}

func (h *Handler) HandleCreateURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	log.Printf("Получен URL в POST запросе: %s", originalURL)

	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortID, err := h.urlService.CreateShortURL(originalURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := "http://localhost:8080/" + shortID
	log.Printf("Создана короткая ссылка: %s", shortURL)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	log.Printf("Получен запрос: метод=%s, путь=%s", r.Method, r.URL.Path)

	shortID := chi.URLParam(r, "shortID")
	log.Printf("Извлечен shortID: %s", shortID)

	if shortID == "" {
		log.Printf("Пустой shortID")
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	log.Printf("Попытка получить оригинальный URL для shortID: %s", shortID)
	originalURL, exists := h.urlService.GetOriginalURL(shortID)
	log.Printf("Результат GetOriginalURL - существует: %v, URL: %s", exists, originalURL)

	if !exists {
		log.Printf("URL не найден для shortID: %s", shortID)
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	log.Printf("Установка заголовка Location: %s", originalURL)
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
)

type Handler struct {
	urlService *service.URLService
}

func NewHandler(urlService *service.URLService) *Handler {
	return &Handler{
		urlService: urlService,
	}
}

func (h *Handler) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
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

func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	log.Printf("Получен запрос: метод=%s, путь=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("Неверный метод запроса: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := strings.TrimPrefix(r.URL.Path, "/")
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

func (h *Handler) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Начало обработки запроса: %s %s", r.Method, r.URL.Path)
	if r.URL.Path == "/" {
		h.handleCreateURL(w, r)
	} else {
		h.handleRedirect(w, r)
	}
}

func main() {
	cfg := config.NewConfig()
	urlService := service.NewURLService()
	handler := NewHandler(urlService)

	http.HandleFunc("/", handler.handleRequest)

	log.Printf("Сервер запускается на http://localhost%s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, nil); err != nil {
		log.Fatal(err)
	}
}

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

// URLService defines the interface for working with URLs
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
			h.logger.Error("Error closing request body", zap.Error(err))
		}
	}()

	originalURL := strings.TrimSpace(string(body))
	h.logger.Info("Received URL in POST request", zap.String("url", originalURL))

	if originalURL == "" {
		http.Error(w, emptyURLMessage, http.StatusBadRequest)
		return
	}

	shortID, err := h.service.CreateShortURL(originalURL)
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

		h.logger.Error("Error in service when creating URL", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Short URL created", zap.String("url", shortURL))
	w.Header().Set("Content-Type", contentTypePlain)
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		h.logger.Error("Error writing response", zap.Error(err))
		return
	}
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Request received", zap.String("method", r.Method), zap.String("path", r.URL.Path))

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get shortID from URL path or parameter
	shortID := chi.URLParam(r, "shortID")
	if shortID == "" {
		shortID = strings.TrimPrefix(r.URL.Path, "/")
	}
	h.logger.Info("Extracted shortID", zap.String("shortID", shortID))

	if shortID == "" {
		h.logger.Warn("Empty shortID")
		http.Error(w, invalidURLMessage, http.StatusBadRequest)
		return
	}

	h.logger.Info("Attempting to get original URL", zap.String("shortID", shortID))
	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		h.logger.Warn("URL not found", zap.String("shortID", shortID), zap.Error(err))
		http.Error(w, urlNotFoundMessage, http.StatusNotFound)
		return
	}

	h.logger.Info("Setting Location header", zap.String("url", originalURL))
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
			h.logger.Error("Error closing request body", zap.Error(err))
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
			h.logger.Info("URL already exists (conflict) in /api/shorten", zap.String("original_url", req.URL), zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", contentTypeJSON)
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Error("Error writing JSON response (conflict)", zap.Error(err))
			}
			return
		}

		h.logger.Error("Error in service when creating URL in /api/shorten", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Short URL created", zap.String("url", shortURL))
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Error writing JSON response", zap.Error(err))
		return
	}
}

// HandleShortenBatch processes requests for batch shortening URLs
func (h *Handler) HandleShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Content-Type (must be application/json)
	// Consider gzip compression, which is handled by middleware
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, contentTypeJSON) {
		// If Content-Type is not application/json but body was successfully decompressed by GzipMiddleware,
		// possibly original Content-Type was application/json
		// GzipMiddleware should have preserved original Content-Type in some header or context?
		// Check standard Accept-Encoding - not it.
		// The easiest way is to rely on the fact that if it's not json, Decode will return an error.
		// Simply log warning if Content-Type is not json
		if !strings.Contains(contentType, "application/json") {
			h.logger.Warn("Request Content-Type is not application/json", zap.String("content-type", contentType))
		}
	}

	var reqBatch []models.BatchRequestEntry // Use model from package models
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
		h.logger.Error("Error decoding batch request JSON", zap.Error(err), zap.ByteString("body", bodyBytes))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Check for empty batch
	if len(reqBatch) == 0 {
		h.logger.Info("Received empty batch request")
		http.Error(w, "Empty batch is not allowed", http.StatusBadRequest)
		return
	}

	// Call service method for batch processing
	respBatch, err := h.service.CreateShortURLsBatch(reqBatch) // Pass []models.BatchRequestEntry
	if err != nil {
		h.logger.Error("Error processing batch in service", zap.Error(err))
		// Determine which error to return to client
		// TODO: Maybe return 400 Bad Request if error is related to invalid data in batch?
		// For now, return 500
		http.Error(w, "Internal server error during batch processing", http.StatusInternalServerError)
		return
	}

	// If service returned empty slice (e.g., all URLs were invalid)
	if len(respBatch) == 0 {
		// Controversial moment: return empty array with code 201 or error 400?
		// Return 400, as fact nothing was created.
		http.Error(w, "All URLs in the batch were invalid or empty", http.StatusBadRequest)
		return
	}

	// Encode response
	respBody, err := json.Marshal(respBatch)
	if err != nil {
		h.logger.Error("Error encoding batch response JSON", zap.Error(err))
		http.Error(w, "Internal server error during response encoding", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated) // Successful creation
	if _, err := w.Write(respBody); err != nil {
		h.logger.Error("Error writing batch response", zap.Error(err))
	}
}

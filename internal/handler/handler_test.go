package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockURLService реализует интерфейс service.URLService для тестов
type mockURLService struct {
	createShortURLFunc       func(ctx context.Context, originalURL string) (string, error)
	getOriginalURLFunc       func(ctx context.Context, shortURL string) (string, error)
	createShortURLsBatchFunc func(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	getStorageFunc           func() storage.URLStorage
	checkConnectionFunc      func(ctx context.Context) error
	getUserURLsFunc          func(ctx context.Context, userID string) ([]models.UserURL, error)
	urls                     map[string]string
	deletedURLs              map[string]bool
}

func (m *mockURLService) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(ctx, originalURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockURLService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(ctx, shortURL)
	}
	if m.deletedURLs[shortURL] {
		return "", storage.ErrURLDeleted
	}
	if url, exists := m.urls[shortURL]; exists {
		return url, nil
	}
	return "", storage.ErrURLNotFound
}

func (m *mockURLService) CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	if m.createShortURLsBatchFunc != nil {
		return m.createShortURLsBatchFunc(ctx, batch)
	}
	return nil, errors.New("not implemented")
}

func (m *mockURLService) GetStorage() storage.URLStorage {
	if m.getStorageFunc != nil {
		return m.getStorageFunc()
	}
	return nil
}

func (m *mockURLService) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return errors.New("not implemented")
}

func (m *mockURLService) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockURLService) BatchDeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	for _, shortURL := range shortURLs {
		m.deletedURLs[shortURL] = true
	}
	return nil
}

// mockDatabaseChecker реализует интерфейсы storage.URLStorage и storage.DatabaseChecker для тестов
type mockDatabaseChecker struct {
	saveFunc                  func(ctx context.Context, shortURL, originalURL, userID string) error
	getFunc                   func(ctx context.Context, shortURL string) (string, error)
	saveBatchFunc             func(ctx context.Context, batch []storage.BatchEntry) error
	getShortURLByOriginalFunc func(ctx context.Context, originalURL string) (string, error)
	checkConnectionFunc       func(ctx context.Context) error
	getUserURLsFunc           func(ctx context.Context, userID string) ([]models.UserURL, error)
}

func (m *mockDatabaseChecker) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, shortURL, originalURL, userID)
	}
	return errors.New("not implemented")
}

func (m *mockDatabaseChecker) Get(ctx context.Context, shortURL string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, shortURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockDatabaseChecker) SaveBatch(ctx context.Context, batch []storage.BatchEntry) error {
	if m.saveBatchFunc != nil {
		return m.saveBatchFunc(ctx, batch)
	}
	return errors.New("not implemented")
}

func (m *mockDatabaseChecker) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	if m.getShortURLByOriginalFunc != nil {
		return m.getShortURLByOriginalFunc(ctx, originalURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockDatabaseChecker) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return errors.New("not implemented")
}

func (m *mockDatabaseChecker) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockDatabaseChecker) BatchDelete(ctx context.Context, shortURLs []string, userID string) error {
	return nil
}

// mockStorage реализует интерфейс storage.URLStorage для тестов
type mockStorage struct {
	saveFunc                  func(ctx context.Context, shortURL, originalURL, userID string) error
	getFunc                   func(ctx context.Context, shortURL string) (string, error)
	saveBatchFunc             func(ctx context.Context, batch []storage.BatchEntry) error
	getShortURLByOriginalFunc func(ctx context.Context, originalURL string) (string, error)
	getUserURLsFunc           func(ctx context.Context, userID string) ([]models.UserURL, error)
}

func (m *mockStorage) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, shortURL, originalURL, userID)
	}
	return errors.New("not implemented")
}

func (m *mockStorage) Get(ctx context.Context, shortURL string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, shortURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockStorage) SaveBatch(ctx context.Context, batch []storage.BatchEntry) error {
	if m.saveBatchFunc != nil {
		return m.saveBatchFunc(ctx, batch)
	}
	return errors.New("not implemented")
}

func (m *mockStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	if m.getShortURLByOriginalFunc != nil {
		return m.getShortURLByOriginalFunc(ctx, originalURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStorage) BatchDelete(ctx context.Context, shortURLs []string, userID string) error {
	return nil
}

func TestHandleCreateURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		mockService    *mockURLService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "Valid URL",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(ctx context.Context, originalURL string) (string, error) {
					return "abc123", nil
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/abc123",
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			contentType:    "text/plain",
			body:           "https://example.com",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Invalid content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           "https://example.com",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid Content-Type",
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "empty URL",
		},
		{
			name:        "Service error",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(ctx context.Context, originalURL string) (string, error) {
					return "", errors.New("service error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			logger, _ := zap.NewDevelopment()
			h := NewHandler(tt.mockService, cfg, logger)

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.HandleCreateURL(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, strings.TrimSpace(w.Body.String()))
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		shortURL       string
		mockService    *mockURLService
		expectedStatus int
		expectedURL    string
	}{
		{
			name:     "Valid short URL",
			method:   http.MethodGet,
			shortURL: "abc123",
			mockService: &mockURLService{
				getOriginalURLFunc: func(ctx context.Context, shortURL string) (string, error) {
					return "https://example.com", nil
				},
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedURL:    "https://example.com",
		},
		{
			name:           "Invalid method",
			method:         http.MethodPost,
			shortURL:       "abc123",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:     "URL not found",
			method:   http.MethodGet,
			shortURL: "nonexistent",
			mockService: &mockURLService{
				getOriginalURLFunc: func(ctx context.Context, shortURL string) (string, error) {
					return "", storage.ErrURLNotFound
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "Service error",
			method:   http.MethodGet,
			shortURL: "abc123",
			mockService: &mockURLService{
				getOriginalURLFunc: func(ctx context.Context, shortURL string) (string, error) {
					return "", errors.New("service error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			logger, _ := zap.NewDevelopment()
			h := NewHandler(tt.mockService, cfg, logger)

			r := chi.NewRouter()
			r.Get("/{id}", h.HandleRedirect)

			req := httptest.NewRequest(tt.method, "/"+tt.shortURL, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedURL != "" {
				assert.Equal(t, tt.expectedURL, w.Header().Get("Location"))
			}
		})
	}
}

func TestHandlePing(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		mockService    *mockURLService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Valid ping",
			method: http.MethodGet,
			mockService: &mockURLService{
				checkConnectionFunc: func(ctx context.Context) error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Invalid method",
			method:         http.MethodPost,
			mockService:    &mockURLService{},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name:   "Connection error",
			method: http.MethodGet,
			mockService: &mockURLService{
				checkConnectionFunc: func(ctx context.Context) error {
					return errors.New("connection error")
				},
			},
			expectedStatus: http.StatusGone,
			expectedBody:   "Storage is no longer available\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			logger, _ := zap.NewDevelopment()
			h := NewHandler(tt.mockService, cfg, logger)

			req := httptest.NewRequest(tt.method, "/ping", nil)
			w := httptest.NewRecorder()

			h.HandlePing(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandleDeleteUserURLs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	mockService := &mockURLService{
		urls:        make(map[string]string),
		deletedURLs: make(map[string]bool),
	}

	handler := NewHandler(mockService, cfg, logger)

	// Создаем тестовый URL
	mockService.urls["test123"] = "https://example.com"

	// Тест успешного удаления
	deleteRequest := models.DeleteRequest{"test123"}
	body, _ := json.Marshal(deleteRequest)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "user123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleDeleteUserURLs(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}

func TestHandleRedirectDeletedURL(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	mockService := &mockURLService{
		urls:        make(map[string]string),
		deletedURLs: make(map[string]bool),
	}

	handler := NewHandler(mockService, cfg, logger)

	// Создаем и удаляем URL
	mockService.urls["test123"] = "https://example.com"
	mockService.deletedURLs["test123"] = true

	req := httptest.NewRequest(http.MethodGet, "/test123", nil)
	w := httptest.NewRecorder()

	handler.HandleRedirect(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("Expected status %d for deleted URL, got %d", http.StatusGone, w.Code)
	}
}

var _ service.URLService = (*mockURLService)(nil)
var _ storage.URLStorage = (*mockStorage)(nil)
var _ storage.URLStorage = (*mockDatabaseChecker)(nil)
var _ storage.DatabaseChecker = (*mockDatabaseChecker)(nil)

// Benchmarks

func setupBenchHandler(b *testing.B) *Handler {
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key",
	}

	urls := make(map[string]string)
	deletedURLs := make(map[string]bool)

	mockSvc := &mockURLService{
		urls:        urls,
		deletedURLs: deletedURLs,
		createShortURLFunc: func(ctx context.Context, originalURL string) (string, error) {
			shortURL := fmt.Sprintf("short-%d", len(urls))
			urls[shortURL] = originalURL
			return shortURL, nil
		},
		getOriginalURLFunc: func(ctx context.Context, shortURL string) (string, error) {
			if deletedURLs[shortURL] {
				return "", storage.ErrURLDeleted
			}
			if url, exists := urls[shortURL]; exists {
				return url, nil
			}
			return "", storage.ErrURLNotFound
		},
		createShortURLsBatchFunc: func(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
			result := make([]models.BatchResponseEntry, len(batch))
			for i, entry := range batch {
				shortURL := fmt.Sprintf("batch-%d-%d", len(urls), i)
				urls[shortURL] = entry.OriginalURL
				result[i] = models.BatchResponseEntry{
					CorrelationID: entry.CorrelationID,
					ShortURL:      cfg.BaseURL + "/" + shortURL,
				}
			}
			return result, nil
		},
	}

	logger := zap.NewNop()
	return NewHandler(mockSvc, cfg, logger)
}

func BenchmarkHandler_HandleCreateURL(b *testing.B) {
	handler := setupBenchHandler(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		originalURL := fmt.Sprintf("https://bench-handler-create-%d.com/path", i)
		body := strings.NewReader(originalURL)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "text/plain")

		// Add user context
		userID := fmt.Sprintf("bench-handler-user-%d", b.N)
		ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleCreateURL(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}

func BenchmarkHandler_HandleShortenURL(b *testing.B) {
	handler := setupBenchHandler(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		originalURL := fmt.Sprintf("https://bench-handler-shorten-%d.com/path", i)
		requestBody := ShortenRequest{URL: originalURL}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Add user context
		userID := fmt.Sprintf("bench-handler-shorten-user-%d", b.N)
		ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleShortenURL(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}

func BenchmarkHandler_HandleRedirect(b *testing.B) {
	handler := setupBenchHandler(b)

	// Pre-populate with URLs
	numEntries := 10000
	shortURLs := make([]string, numEntries)
	mockSvc := handler.service.(*mockURLService)
	for i := 0; i < numEntries; i++ {
		originalURL := fmt.Sprintf("https://bench-handler-redirect-%d.com/path", i)
		shortURL := fmt.Sprintf("bench-short-%d", i)
		mockSvc.urls[shortURL] = originalURL
		shortURLs[i] = shortURL
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shortURL := shortURLs[i%numEntries]
		req := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)

		// Add chi URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("shortURL", shortURL)
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleRedirect(w, req)

		if w.Code != http.StatusTemporaryRedirect {
			b.Fatalf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
		}
	}
}

func BenchmarkHandler_HandleShortenBatch(b *testing.B) {
	handler := setupBenchHandler(b)

	// Test different batch sizes
	batchSizes := []int{10, 50, 100}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := make([]models.BatchRequestEntry, batchSize)
				baseID := i*1000000 + batchSize*1000 // Ensure uniqueness across iterations
				for j := 0; j < batchSize; j++ {
					batch[j] = models.BatchRequestEntry{
						CorrelationID: fmt.Sprintf("corr_%d_%d", baseID, j),
						OriginalURL:   fmt.Sprintf("https://bench-handler-batch-%d-%d.com/path", baseID, j),
					}
				}

				body, _ := json.Marshal(batch)
				req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				// Add user context
				userID := fmt.Sprintf("bench-handler-batch-user-%d", b.N)
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
				req = req.WithContext(ctx)

				w := httptest.NewRecorder()
				handler.HandleShortenBatch(w, req)

				if w.Code != http.StatusCreated {
					b.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
				}
			}
		})
	}
}

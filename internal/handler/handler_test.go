package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
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
	return "", errors.New("not implemented")
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

// mockDatabaseChecker реализует интерфейсы storage.URLStorage и storage.DatabaseChecker для тестов
type mockDatabaseChecker struct {
	saveFunc                  func(ctx context.Context, shortURL, originalURL string) error
	getFunc                   func(ctx context.Context, shortURL string) (string, error)
	saveBatchFunc             func(ctx context.Context, batch []storage.BatchEntry) error
	getShortURLByOriginalFunc func(ctx context.Context, originalURL string) (string, error)
	checkConnectionFunc       func(ctx context.Context) error
}

func (m *mockDatabaseChecker) Save(ctx context.Context, shortURL, originalURL string) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, shortURL, originalURL)
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

// mockStorage реализует интерфейс storage.URLStorage для тестов
type mockStorage struct {
	saveFunc                  func(ctx context.Context, shortURL, originalURL string) error
	getFunc                   func(ctx context.Context, shortURL string) (string, error)
	saveBatchFunc             func(ctx context.Context, batch []storage.BatchEntry) error
	getShortURLByOriginalFunc func(ctx context.Context, originalURL string) (string, error)
}

func (m *mockStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, shortURL, originalURL)
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
		},
		{
			name:           "Invalid method",
			method:         http.MethodPost,
			mockService:    &mockURLService{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "Connection error",
			method: http.MethodGet,
			mockService: &mockURLService{
				checkConnectionFunc: func(ctx context.Context) error {
					return errors.New("connection error")
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

			req := httptest.NewRequest(tt.method, "/ping", nil)
			w := httptest.NewRecorder()

			h.HandlePing(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

var _ service.URLService = (*mockURLService)(nil)
var _ storage.URLStorage = (*mockStorage)(nil)
var _ storage.URLStorage = (*mockDatabaseChecker)(nil)
var _ storage.DatabaseChecker = (*mockDatabaseChecker)(nil)

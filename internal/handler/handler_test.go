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
	createShortURLFunc       func(ctx context.Context, originalURL string, userID string) (string, error)
	getOriginalURLFunc       func(ctx context.Context, shortURL string) (string, error)
	getUserURLsFunc          func(ctx context.Context) ([]models.UserURL, error)
	createShortURLsBatchFunc func(ctx context.Context, batch []service.BatchRequest, userID string) ([]service.BatchResponse, error)
	PingFunc                 func(ctx context.Context) error
}

func (m *mockURLService) CreateShortURL(ctx context.Context, originalURL string, userID string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(ctx, originalURL, userID)
	}
	return "", nil
}

func (m *mockURLService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(ctx, shortURL)
	}
	return "", nil
}

func (m *mockURLService) GetUserURLs(ctx context.Context) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx)
	}
	return nil, nil
}

func (m *mockURLService) CreateShortURLsBatch(ctx context.Context, batch []service.BatchRequest, userID string) ([]service.BatchResponse, error) {
	if m.createShortURLsBatchFunc != nil {
		return m.createShortURLsBatchFunc(ctx, batch, userID)
	}
	return nil, nil
}

func (m *mockURLService) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

// mockStorage реализует интерфейс storage.URLStorage для тестов
type mockStorage struct {
	storage.URLStorage
	saveURLFunc         func(ctx context.Context, shortURL, originalURL string) error
	getOriginalURLFunc  func(ctx context.Context, shortURL string) (string, error)
	getShortURLFunc     func(ctx context.Context, originalURL string) (string, error)
	getUserURLsFunc     func(ctx context.Context, userID string) ([]models.UserURL, error)
	checkConnectionFunc func(ctx context.Context) error
	closeFunc           func() error
}

func (m *mockStorage) SaveURL(ctx context.Context, shortURL, originalURL string) error {
	if m.saveURLFunc != nil {
		return m.saveURLFunc(ctx, shortURL, originalURL)
	}
	return nil
}

func (m *mockStorage) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(ctx, shortURL)
	}
	return "", nil
}

func (m *mockStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	if m.getShortURLFunc != nil {
		return m.getShortURLFunc(ctx, originalURL)
	}
	return "", nil
}

func (m *mockStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockStorage) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return nil
}

func (m *mockStorage) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockDatabaseChecker реализует интерфейс storage.URLStorage для тестов
type mockDatabaseChecker struct {
	storage.URLStorage
	checkConnectionFunc func(ctx context.Context) error
}

func (m *mockDatabaseChecker) SaveURL(ctx context.Context, shortURL, originalURL string) error {
	return nil
}

func (m *mockDatabaseChecker) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	return "", nil
}

func (m *mockDatabaseChecker) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	return "", nil
}

func (m *mockDatabaseChecker) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	return nil, nil
}

func (m *mockDatabaseChecker) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return nil
}

func (m *mockDatabaseChecker) Close() error {
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
				createShortURLFunc: func(ctx context.Context, originalURL string, userID string) (string, error) {
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
			expectedBody:   "empty request body",
		},
		{
			name:        "Service error",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(ctx context.Context, originalURL string, userID string) (string, error) {
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
				PingFunc: func(ctx context.Context) error {
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
				PingFunc: func(ctx context.Context) error {
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

var _ service.URLService = (*mockURLService)(nil)
var _ storage.URLStorage = (*mockStorage)(nil)
var _ storage.URLStorage = (*mockDatabaseChecker)(nil)
var _ storage.DatabaseChecker = (*mockDatabaseChecker)(nil)

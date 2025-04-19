package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/go-chi/chi/v5"
)

type mockURLService struct {
	createShortURLFunc func(url string) (string, error)
	getOriginalURLFunc func(shortID string) (string, bool)
}

func (m *mockURLService) CreateShortURL(url string) (string, error) {
	if m.createShortURLFunc == nil {
		panic("createShortURLFunc is nil in mockURLService")
	}
	return m.createShortURLFunc(url)
}

func (m *mockURLService) GetOriginalURL(shortID string) (string, bool) {
	if m.getOriginalURLFunc == nil {
		panic("getOriginalURLFunc is nil in mockURLService")
	}
	return m.getOriginalURLFunc(shortID)
}

func TestHandleCreateURL(t *testing.T) {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		mockService    URLService
	}{
		{
			name:           "Valid POST request",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
			mockService: &mockURLService{
				createShortURLFunc: func(url string) (string, error) {
					return "abc123", nil
				},
			},
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusMethodNotAllowed,
			mockService:    &mockURLService{},
		},
		{
			name:           "Invalid content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           "https://example.com",
			expectedStatus: http.StatusBadRequest,
			mockService:    &mockURLService{},
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			mockService:    &mockURLService{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h := NewHandler(tt.mockService, cfg)
			h.HandleCreateURL(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Дополнительная проверка тела ответа для успешного случая
			if tt.name == "Valid POST request" && w.Code == http.StatusCreated {
				expectedBody := cfg.BaseURL + "/abc123"
				if body := w.Body.String(); body != expectedBody {
					t.Errorf("handler returned wrong body: got %v want %v",
						body, expectedBody)
				}
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		mockService    URLService
	}{
		{
			name:           "Valid redirect",
			path:           "/abc123",
			expectedStatus: http.StatusTemporaryRedirect,
			mockService: &mockURLService{
				getOriginalURLFunc: func(shortID string) (string, bool) {
					return "https://example.com", true
				},
			},
		},
		{
			name:           "Invalid method",
			path:           "/abc123",
			expectedStatus: http.StatusMethodNotAllowed,
			mockService:    &mockURLService{},
		},
		{
			name:           "URL not found",
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			mockService: &mockURLService{
				getOriginalURLFunc: func(shortID string) (string, bool) {
					return "", false
				},
			},
		},
		{
			name:           "Empty short ID in path",
			path:           "/",
			expectedStatus: http.StatusNotFound,
			mockService:    &mockURLService{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			h := NewHandler(tt.mockService, cfg)
			r.Get("/{shortID}", h.HandleRedirect)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

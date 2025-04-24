package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
)

type mockURLService struct {
	createShortURLFunc func(url string) (string, error)
	getOriginalURLFunc func(shortID string) (string, error)
}

func (m *mockURLService) CreateShortURL(url string) (string, error) {
	return m.createShortURLFunc(url)
}

func (m *mockURLService) GetOriginalURL(shortID string) (string, error) {
	return m.getOriginalURLFunc(shortID)
}

func TestHandleCreateURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		mockService    service.URLService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "Valid URL",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(url string) (string, error) {
					return "abc123", nil
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/abc123",
		},
		{
			name:           "Invalid Method",
			method:         http.MethodGet,
			contentType:    "text/plain",
			body:           "https://example.com",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name:           "Invalid Content-Type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           "https://example.com",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid Content-Type\n",
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty URL\n",
		},
		{
			name:        "Service Error",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(url string) (string, error) {
					return "", errors.New("service error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			h := NewHandler(tt.mockService, cfg)

			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			h.HandleCreateURL(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	tests := []struct {
		name           string
		shortID        string
		mockService    service.URLService
		expectedStatus int
		expectedURL    string
	}{
		{
			name:    "Valid ShortID",
			shortID: "abc123",
			mockService: &mockURLService{
				getOriginalURLFunc: func(shortID string) (string, error) {
					return "https://example.com", nil
				},
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedURL:    "https://example.com",
		},
		{
			name:           "Empty ShortID",
			shortID:        "",
			mockService:    &mockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
		{
			name:    "URL Not Found",
			shortID: "abc123",
			mockService: &mockURLService{
				getOriginalURLFunc: func(shortID string) (string, error) {
					return "", errors.New("URL not found")
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			h := NewHandler(tt.mockService, cfg)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortID, nil)
			w := httptest.NewRecorder()

			// Создаем контекст с параметром shortID
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("shortID", tt.shortID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.HandleRedirect(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedURL != "" && w.Header().Get("Location") != tt.expectedURL {
				t.Errorf("expected Location header %q, got %q", tt.expectedURL, w.Header().Get("Location"))
			}
		})
	}
}

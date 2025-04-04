package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/service"
)

type mockURLService struct {
	createShortURLFunc func(url string) (string, error)
	getOriginalURLFunc func(shortID string) (string, bool)
}

func (m *mockURLService) CreateShortURL(url string) (string, error) {
	return m.createShortURLFunc(url)
}

func (m *mockURLService) GetOriginalURL(shortID string) (string, bool) {
	return m.getOriginalURLFunc(shortID)
}

func TestHandleCreateURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		mockService    service.URLService
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
			rr := httptest.NewRecorder()

			handler := NewHandler(tt.mockService)
			handler.HandleCreateURL(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		mockService    service.URLService
	}{
		{
			name:           "Valid redirect",
			method:         http.MethodGet,
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
			method:         http.MethodPost,
			path:           "/abc123",
			expectedStatus: http.StatusMethodNotAllowed,
			mockService:    &mockURLService{},
		},
		{
			name:           "URL not found",
			method:         http.MethodGet,
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			mockService: &mockURLService{
				getOriginalURLFunc: func(shortID string) (string, bool) {
					return "", false
				},
			},
		},
		{
			name:           "Empty short ID",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			mockService:    &mockURLService{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler := NewHandler(tt.mockService)
			handler.HandleRedirect(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}

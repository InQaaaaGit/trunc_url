package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockURLService struct {
	urls map[string]string
}

func (m *mockURLService) CreateShortURL(url string) (string, error) {
	return "testID", nil
}

func (m *mockURLService) GetOriginalURL(shortID string) (string, bool) {
	url, exists := m.urls[shortID]
	return url, exists
}

func TestHandleShortenURL(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedResult string
	}{
		{
			name: "Valid URL",
			requestBody: map[string]string{
				"url": "https://practicum.yandex.ru",
			},
			expectedStatus: http.StatusCreated,
			expectedResult: "http://localhost:8080/testID",
		},
		{
			name: "Empty URL",
			requestBody: map[string]string{
				"url": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&mockURLService{}, cfg)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleShortenURL(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response ShortenResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, response.Result)
			}
		})
	}
}

func TestHandleShortenURLInvalidMethod(t *testing.T) {
	handler := NewHandler(&mockURLService{}, &config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/shorten", nil)
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleShortenURLInvalidContentType(t *testing.T) {
	handler := NewHandler(&mockURLService{}, &config.Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid POST request",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/testID",
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			contentType:    "text/plain",
			body:           "https://example.com",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           "https://example.com",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty URL",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&mockURLService{}, cfg)

			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.HandleCreateURL(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "Valid redirect",
			method:           http.MethodGet,
			path:             "/testID",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://example.com",
		},
		{
			name:           "Invalid method",
			method:         http.MethodPost,
			path:           "/testID",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "URL not found",
			method:         http.MethodGet,
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty short ID",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
		},
	}

	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockURLService{
				urls: map[string]string{
					"testID": "https://example.com",
				},
			}
			handler := NewHandler(mockService, cfg)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := chi.NewRouteContext()
			ctx.URLParams.Add("shortID", tt.path[1:])
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

			w := httptest.NewRecorder()

			handler.HandleRedirect(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.expectedLocation, w.Header().Get("Location"))
			}
		})
	}
}

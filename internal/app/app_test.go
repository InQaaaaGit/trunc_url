package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)
	assert.NotNil(t, app.config)
	assert.NotNil(t, app.router)
	assert.NotNil(t, app.logger)
	assert.NotNil(t, app.handler)
}

func TestApp_Configure(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	err = app.Configure()
	assert.NoError(t, err)
}

func TestApp_GetServer(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	server := app.GetServer()
	assert.NotNil(t, server)
	assert.Equal(t, cfg.ServerAddress, server.Addr)
	assert.Equal(t, 10*time.Second, server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.WriteTimeout)
	assert.Equal(t, 120*time.Second, server.IdleTimeout)
}

func TestApp_SetupRoutes(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// Test that routes are properly configured after setupRoutes
	app.setupRoutes()

	// Test POST / route
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)

	// Test GET /ping route
	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr = httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)

	// Test POST /api/shorten route
	req = httptest.NewRequest(http.MethodPost, "/api/shorten", nil)
	rr = httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)

	// Test POST /api/shorten/batch route
	req = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", nil)
	rr = httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)

	// Test GET /api/user/urls route
	req = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rr = httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)

	// Test DELETE /api/user/urls route
	req = httptest.NewRequest(http.MethodDelete, "/api/user/urls", nil)
	rr = httptest.NewRecorder()
	app.router.ServeHTTP(rr, req)
	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)
}

func TestNewApp_WithInvalidConfig(t *testing.T) {
	// Test with config that might cause service creation to fail
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/invalid/path/to/nonexistent/directory/file.json",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	// This should still work because service falls back to memory storage
	app, err := NewApp(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, app)
}

func TestApp_MiddlewareIntegration(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	app.setupRoutes()

	// Test that middleware is applied by checking headers
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	app.router.ServeHTTP(rr, req)

	// Check that user_id cookie is set (AuthMiddleware)
	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	var userIDCookieFound bool
	for _, cookie := range cookies {
		if cookie.Name == "user_id" {
			userIDCookieFound = true
			break
		}
	}
	assert.True(t, userIDCookieFound, "AuthMiddleware should set user_id cookie")
}

func TestApp_RunContext(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":0", // Use random available port
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// Test server creation and immediate shutdown
	server := app.GetServer()
	server.Addr = ":0" // Use random port

	// Start server in goroutine
	go func() {
		server.ListenAndServe()
	}()

	// Immediately shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestApp_InternalStatsEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		trustedSubnet  string
		clientIP       string
		xRealIP        string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "no trusted subnet configured",
			trustedSubnet:  "",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "valid IP in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"urls":  float64(0), // JSON numbers are unmarshaled as float64
				"users": float64(0),
			},
		},
		{
			name:           "valid IP in trusted subnet using X-Real-IP",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "127.0.0.1",
			xRealIP:        "192.168.1.100",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"urls":  float64(0),
				"users": float64(0),
			},
		},
		{
			name:           "IP not in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "10.0.0.100",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем конфигурацию
			cfg := &config.Config{
				TrustedSubnet: tt.trustedSubnet,
			}

			// Создаем приложение
			app, err := NewApp(cfg)
			assert.NoError(t, err)
			assert.NotNil(t, app)

			// Настраиваем маршруты
			app.setupRoutes()

			// Создаем запрос
			req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
			req.RemoteAddr = tt.clientIP + ":12345"
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			// Создаем response recorder
			w := httptest.NewRecorder()

			// Выполняем запрос
			app.router.ServeHTTP(w, req)

			// Проверяем статус
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Если ожидаем успешный ответ, проверяем тело
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}
		})
	}
}

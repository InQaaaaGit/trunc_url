package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Сохраняем оригинальные переменные окружения
	originalEnv := make(map[string]string)
	for _, env := range []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN"} {
		if value, exists := os.LookupEnv(env); exists {
			originalEnv[env] = value
		}
	}

	// Устанавливаем тестовые значения
	os.Setenv("SERVER_ADDRESS", ":8080")
	os.Setenv("BASE_URL", "http://localhost:8080")
	os.Setenv("FILE_STORAGE_PATH", "")
	os.Setenv("DATABASE_DSN", "")

	// Запускаем тесты
	code := m.Run()

	// Восстанавливаем оригинальные значения
	for env, value := range originalEnv {
		os.Setenv(env, value)
	}
	for _, env := range []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN"} {
		if _, exists := originalEnv[env]; !exists {
			os.Unsetenv(env)
		}
	}

	os.Exit(code)
}

func TestMainFunction(t *testing.T) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Запускаем main в отдельной горутине
	go func() {
		main()
	}()

	// Ждем завершения контекста
	<-ctx.Done()
}

func TestGetConfig(t *testing.T) {
	// Тест с дефолтными значениями
	cfg := getConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, ":8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Empty(t, cfg.FileStoragePath)
	assert.Empty(t, cfg.DatabaseDSN)

	// Тест с пользовательскими значениями
	os.Setenv("SERVER_ADDRESS", ":9090")
	os.Setenv("BASE_URL", "http://localhost:9090")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/urls.json")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/db")

	cfg = getConfig()
	assert.Equal(t, ":9090", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:9090", cfg.BaseURL)
	assert.Equal(t, "/tmp/urls.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DatabaseDSN)
}

func TestSetupLogger(t *testing.T) {
	logger, err := setupLogger()
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestRunServer(t *testing.T) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Запускаем сервер в отдельной горутине
	go func() {
		err := runServer(ctx)
		assert.Error(t, err) // Ожидаем ошибку из-за таймаута
	}()

	// Ждем завершения контекста
	<-ctx.Done()
}

func TestAPIEndpoints(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	appInstance, err := app.NewApp(cfg)
	require.NoError(t, err)

	// Создаем тестовый сервер
	ts := httptest.NewServer(appInstance.Router())
	defer ts.Close()

	// Создаем тестовые запросы
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		headers    map[string]string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "POST /api/shorten - пустой запрос",
			method:     http.MethodPost,
			path:       "/api/shorten",
			body:       "",
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "empty request body",
		},
		{
			name:       "POST /api/shorten - неверный формат JSON",
			method:     http.MethodPost,
			path:       "/api/shorten",
			body:       `{"url":}`,
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid request format",
		},
		{
			name:       "POST /api/shorten - неверный Content-Type",
			method:     http.MethodPost,
			path:       "/api/shorten",
			body:       `{"url":"http://example.com"}`,
			headers:    map[string]string{"Content-Type": "text/plain"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid content type",
		},
		{
			name:       "POST /api/shorten/batch - пустой запрос",
			method:     http.MethodPost,
			path:       "/api/shorten/batch",
			body:       "",
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "empty request body",
		},
		{
			name:       "GET /api/user/urls - без токена",
			method:     http.MethodGet,
			path:       "/api/user/urls",
			body:       "",
			headers:    map[string]string{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, strings.NewReader(tt.body))
			require.NoError(t, err)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), tt.wantBody)
			}
		})
	}
}

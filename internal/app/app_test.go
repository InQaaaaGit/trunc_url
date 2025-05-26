package app

import (
	"context"
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
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)
	assert.NotNil(t, app.router)
	assert.NotNil(t, app.logger)
	assert.NotNil(t, app.handler)
}

func TestAppRun(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":0", // Используем порт 0 для автоматического выбора свободного порта
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// Запускаем сервер в отдельной горутине
	go func() {
		err := app.Run()
		assert.NoError(t, err)
	}()

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сервер отвечает
	server := app.GetServer()
	assert.NotNil(t, server)
	assert.Equal(t, cfg.ServerAddress, server.Addr)
	assert.NotNil(t, server.Handler)
}

func TestAppConfigure(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	err = app.Configure()
	assert.NoError(t, err)
	assert.NotNil(t, app.handler)
}

func TestAppRoutes(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	// Создаем тестовые запросы для проверки маршрутов
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"POST /", http.MethodPost, "/", http.StatusMethodNotAllowed},
		{"GET /{id}", http.MethodGet, "/abc123", http.StatusTemporaryRedirect},
		{"POST /api/shorten", http.MethodPost, "/api/shorten", http.StatusMethodNotAllowed},
		{"POST /api/shorten/batch", http.MethodPost, "/api/shorten/batch", http.StatusMethodNotAllowed},
		{"GET /api/user/urls", http.MethodGet, "/api/user/urls", http.StatusUnauthorized},
		{"GET /ping", http.MethodGet, "/ping", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			app.router.ServeHTTP(rr, req)
			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestAppWithContext(t *testing.T) {
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	app, err := NewApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	server := app.GetServer()
	go func() {
		err := server.ListenAndServe()
		assert.Error(t, err)
	}()

	<-ctx.Done()
}

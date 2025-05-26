package main

import (
	"context"
	"os"
	"testing"
	"time"

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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"go.uber.org/zap"
)

// getConfig возвращает конфигурацию приложения
func getConfig() *config.Config {
	return &config.Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", ":8080"),
		BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
		FileStoragePath: getEnv("FILE_STORAGE_PATH", ""),
		DatabaseDSN:     getEnv("DATABASE_DSN", ""),
	}
}

// getEnv возвращает значение переменной окружения или дефолтное значение
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// setupLogger создает и настраивает логгер
func setupLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

// runServer запускает HTTP сервер
func runServer(ctx context.Context) error {
	cfg := getConfig()
	logger, err := setupLogger()
	if err != nil {
		return fmt.Errorf("error creating logger: %w", err)
	}

	application, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("error creating application: %w", err)
	}

	server := application.GetServer()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Error starting server", zap.Error(err))
		}
	}()

	// Ожидаем сигнал завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down server")
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down server", zap.String("signal", sig.String()))
	}

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := runServer(ctx); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

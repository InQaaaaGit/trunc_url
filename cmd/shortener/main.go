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
	"go.uber.org/zap/zapcore"
)

// getConfig возвращает конфигурацию приложения
func getConfig() *config.Config {
	return &config.Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", ":8080"),
		BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
		FileStoragePath: getEnv("FILE_STORAGE_PATH", ""), // Для iteration1 это должно быть пусто, чтобы использовался MemoryStorage
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

// setupLogger создает и настраивает логгер для записи в файл
func setupLogger() (*zap.Logger, error) {
	logFilePath := "server_log.txt"
	_ = os.Remove(logFilePath) // Удаляем старый лог-файл при запуске

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file for zap: %w", err)
	}

	core := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zap.InfoLevel)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

// runServer запускает HTTP сервер
func runServer(ctx context.Context, logger *zap.Logger) error {
	logger.Info("runServer called")
	cfg := getConfig()

	application, err := app.NewApp(cfg, logger) // Передаем logger
	if err != nil {
		logger.Error("Error creating application in runServer", zap.Error(err))
		return fmt.Errorf("error creating application: %w", err)
	}

	server := application.GetServer()
	logger.Info("Server configured", zap.String("address", server.Addr))

	errChan := make(chan error, 1)
	go func() {
		logger.Info("Starting server ListenAndServe", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error during ListenAndServe", zap.Error(err))
			errChan <- err
		} else if err == http.ErrServerClosed {
			logger.Info("ListenAndServe: server closed normally")
		} else {
			logger.Info("ListenAndServe: exited without error")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("Server startup/runtime error", zap.Error(err))
		return fmt.Errorf("server startup/runtime error: %w", err)
	case sig := <-sigChan:
		logger.Info("Received OS signal, shutting down server", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Увеличим таймаут для шатдауна
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
		return fmt.Errorf("error shutting down server: %w", err)
	}
	logger.Info("Server shutdown gracefully")
	return nil
}

func main() {
	// Настройка стандартного логгера (на всякий случай, если zap упадет до инициализации)
	stdLogFile, errStdLog := os.OpenFile("server_std_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if errStdLog == nil {
		log.SetOutput(stdLogFile)
	} else {
		fmt.Printf("Failed to open server_std_log.txt: %v\n", errStdLog)
	}
	log.Println("Main function started (standard log)")

	appLogger, err := setupLogger() // Настраиваем zap логгер для записи в server_log.txt
	if err != nil {
		log.Fatalf("CRITICAL: Failed to create zap logger: %v", err)
	}
	defer appLogger.Sync()

	appLogger.Info("Zap logger initialized in main")

	ctx, cancelCtx := context.WithCancel(context.Background()) // Контекст для runServer
	defer cancelCtx()                                          // На случай, если main завершится раньше runServer по другой причине

	if err := runServer(ctx, appLogger); err != nil {
		appLogger.Error("runServer returned an error", zap.Error(err))
		// os.Exit(1) // Не используем Fatal здесь, чтобы успел сработать defer appLogger.Sync()
	} else {
		appLogger.Info("runServer completed without error")
	}
	appLogger.Info("Main function finished")
}

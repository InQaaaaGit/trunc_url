package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/buildinfo"
	"github.com/InQaaaaGit/trunc_url.git/internal/server"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	// Создаем и выводим информацию о сборке
	buildInfo := buildinfo.NewInfo(buildVersion, buildDate, buildCommit)
	buildInfo.Print()

	// Инициализация логгера
	logger, cleanup := server.InitLogger()
	defer cleanup()

	// Инициализация конфигурации
	cfg := server.InitConfig(logger)

	// Создание и настройка приложения
	application, err := app.NewApp(cfg)
	if err != nil {
		logger.Fatal("Ошибка создания приложения", zap.Error(err))
	}
	if err := application.Configure(); err != nil {
		logger.Fatal("Ошибка конфигурации приложения", zap.Error(err))
	}

	// Создание сервера
	httpServer := server.NewHTTPServer(application.GetServer(), cfg, logger)

	// Канал для получения сигналов операционной системы
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Запуск сервера в горутине
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Starting server...")
		if err := httpServer.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Ожидание сигнала завершения или ошибки сервера
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case err := <-serverErr:
		logger.Error("Server error", zap.Error(err))
		return
	}

	// Graceful shutdown
	logger.Info("Initiating graceful shutdown...")

	// Создаем контекст с тайм-аутом для завершения работы
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Завершаем работу сервера
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
	} else {
		logger.Info("Server shutdown completed successfully")
	}

	// Закрываем приложение и сохраняем данные
	if err := application.Close(); err != nil {
		logger.Error("Error closing application", zap.Error(err))
	} else {
		logger.Info("Application resources closed successfully")
	}
}

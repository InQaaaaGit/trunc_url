package main

import (
	"fmt"
	"log"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

// printBuildInfo выводит информацию о сборке приложения
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {
	// Выводим информацию о сборке
	printBuildInfo()

	// Инициализация логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Ошибка синхронизации логгера: %v", err)
		}
	}()

	// Инициализация конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatal("Ошибка инициализации конфигурации", zap.Error(err))
	}

	// Создание и настройка приложения
	application, err := app.NewApp(cfg)
	if err != nil {
		logger.Fatal("Ошибка создания приложения", zap.Error(err))
	}
	if err := application.Configure(); err != nil {
		logger.Fatal("Ошибка конфигурации приложения", zap.Error(err))
	}

	// Запуск сервера
	if cfg.IsHTTPSEnabled() {
		server := application.GetServer()
		logger.Info("Запуск HTTPS сервера",
			zap.String("address", cfg.ServerAddress),
			zap.String("cert", cfg.TLSCertFile),
			zap.String("key", cfg.TLSKeyFile))
		if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			logger.Fatal("HTTPS server failed to start", zap.Error(err))
		}
	} else {
		server := application.GetServer()
		logger.Info("Запуск HTTP сервера", zap.String("address", cfg.ServerAddress))
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal("HTTP server failed to start", zap.Error(err))
		}
	}
}

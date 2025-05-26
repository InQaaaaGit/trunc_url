package main

import (
	"log"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"go.uber.org/zap"
)

func main() {
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
	server := application.GetServer()
	logger.Info("Сервер запускается", zap.String("address", cfg.ServerAddress))
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

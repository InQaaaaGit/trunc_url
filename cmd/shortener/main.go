package main

import (
	"log"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
)

func main() {
	// Инициализация конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}

	// Создание и настройка приложения
	application := app.New(cfg)
	application.Configure()

	// Запуск сервера
	server := application.GetServer()
	log.Printf("Сервер запускается на %s\n", cfg.ServerAddress)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

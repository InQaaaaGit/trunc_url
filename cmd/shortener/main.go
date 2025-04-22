package main

import (
	"log"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/config"
)

func main() {
	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Создание и настройка приложения
	application := app.New(cfg)
	application.Configure()

	// Запуск сервера
	server := application.GetServer()
	log.Printf("Сервер запускается на %s\n", cfg.ServerAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Инициализация логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Инициализация сервисов и обработчиков
	urlService := service.NewURLService()
	handler := handler.NewHandler(urlService, cfg.BaseURL)

	// Создание роутера
	r := chi.NewRouter()

	// Применение middleware логирования
	r.Use(middleware.LoggerMiddleware(logger))

	// Регистрация маршрутов
	r.Post("/", handler.HandleCreateURL)
	r.Get("/{shortID}", handler.HandleRedirect)

	// Запуск сервера
	logger.Info("Starting server", zap.String("address", cfg.ServerAddress))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

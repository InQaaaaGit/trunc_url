package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
)

func main() {
	// Инициализация конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Инициализация логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Error syncing logger: %v", err)
		}
	}()

	// Создание сервиса
	service, err := service.NewURLService(cfg, logger)
	if err != nil {
		logger.Fatal("Error creating service", zap.Error(err))
	}

	// Создание обработчика
	handler := handler.NewHandler(service, cfg, logger)

	// Настройка маршрутизатора
	r := chi.NewRouter()

	// Middleware
	r.Use(handler.WithLogging)
	r.Use(handler.WithGzip)

	// Маршруты
	r.Post("/", handler.HandleCreateURL)
	r.Get("/{shortID}", handler.HandleRedirect)
	r.Post("/api/shorten", handler.HandleShortenURL)
	r.Post("/api/shorten/batch", handler.HandleShortenBatch)
	r.Get("/ping", handler.HandlePing)

	// Запуск сервера
	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("Starting server", zap.String("address", cfg.ServerAddress))
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

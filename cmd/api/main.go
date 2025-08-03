package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
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

	if cfg.IsHTTPSEnabled() {
		logger.Info("Starting HTTPS server",
			zap.String("address", cfg.ServerAddress),
			zap.String("cert", cfg.TLSCertFile),
			zap.String("key", cfg.TLSKeyFile))
		if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			logger.Fatal("HTTPS server error", zap.Error(err))
		}
	} else {
		logger.Info("Starting HTTP server", zap.String("address", cfg.ServerAddress))
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}
}

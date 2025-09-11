package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/InQaaaaGit/trunc_url.git/internal/buildinfo"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/server"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
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

	// Создание сервиса
	urlService, err := service.NewURLService(cfg, logger)
	if err != nil {
		logger.Fatal("Error creating service", zap.Error(err))
	}

	// Создание обработчика
	h := handler.NewHandler(urlService, cfg, logger)

	// Настройка маршрутизатора
	r := chi.NewRouter()

	// Middleware
	r.Use(h.WithLogging)
	r.Use(h.WithGzip)

	// Маршруты
	r.Post("/", h.HandleCreateURL)
	r.Get("/{shortID}", h.HandleRedirect)
	r.Post("/api/shorten", h.HandleShortenURL)
	r.Post("/api/shorten/batch", h.HandleShortenBatch)
	r.Get("/ping", h.HandlePing)

	// Создание HTTP сервера
	httpServer := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Создание и запуск сервера
	serverWrapper := server.NewHTTPServer(httpServer, cfg, logger)
	if err := serverWrapper.Start(); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

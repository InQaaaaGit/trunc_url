package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// App представляет основное приложение
type App struct {
	config  *config.Config
	router  *chi.Mux
	logger  *zap.Logger
	handler *handler.Handler
}

// NewApp создает новый экземпляр приложения
func NewApp(cfg *config.Config) (*App, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	service, err := service.NewURLService(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating service: %w", err)
	}

	handler := handler.NewHandler(service, cfg, logger)

	return &App{
		config:  cfg,
		router:  chi.NewRouter(),
		logger:  logger,
		handler: handler,
	}, nil
}

// Run запускает приложение
func (a *App) Run() error {
	a.setupRoutes()

	server := &http.Server{
		Addr:         a.config.ServerAddress,
		Handler:      a.router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	a.logger.Info("Starting server", zap.String("address", a.config.ServerAddress))
	return server.ListenAndServe()
}

// setupRoutes настраивает маршруты приложения
func (a *App) setupRoutes() {
	// Middleware
	a.router.Use(a.handler.WithLogging)
	a.router.Use(a.handler.WithGzip)
	a.router.Use(middleware.WithAuth)

	// Routes
	a.router.Post("/", a.handler.HandleCreateURL)
	a.router.Get("/{id}", a.handler.HandleRedirect)
	a.router.Post("/api/shorten", a.handler.HandleShortenURL)
	a.router.Post("/api/shorten/batch", a.handler.HandleShortenBatch)
	a.router.Get("/api/user/urls", a.handler.HandleGetUserURLs)
	a.router.Get("/ping", a.handler.HandlePing)
}

// Configure настраивает все слои приложения
func (a *App) Configure() error {
	// Инициализация сервисов и обработчиков
	urlService, err := service.NewURLService(a.config, a.logger)
	if err != nil {
		return err
	}
	handler := handler.NewHandler(urlService, a.config, a.logger)

	// Подключаем middleware
	a.router.Use(handler.WithLogging)
	a.router.Use(handler.WithGzip)

	// Регистрация маршрутов
	a.router.Post("/", handler.HandleCreateURL)
	a.router.Post("/api/shorten", handler.HandleShortenURL)
	a.router.Post("/api/shorten/batch", handler.HandleShortenBatch)
	a.router.Get("/{shortID}", handler.HandleRedirect)

	// Добавляем хендлер для проверки доступности БД
	a.router.Get("/ping", handler.HandlePing)

	return nil
}

// GetServer возвращает настроенный HTTP сервер
func (a *App) GetServer() *http.Server {
	return &http.Server{
		Addr:         a.config.ServerAddress,
		Handler:      a.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

package app

import (
	// "fmt" // Убрано
	// "log" // Убрано
	"fmt"
	"net/http"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
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
func NewApp(cfg *config.Config, logger *zap.Logger) (*App, error) {
	logger.Info("NewApp called")

	service, err := service.NewURLService(cfg, logger) // Используем переданный logger
	if err != nil {
		logger.Error("NewApp: error creating service", zap.Error(err))
		return nil, fmt.Errorf("error creating service: %w", err)
	}
	logger.Info("NewApp: service created")

	handler := handler.NewHandler(service, cfg, logger) // Используем переданный logger
	logger.Info("NewApp: handler created")

	app := &App{
		config:  cfg,
		router:  chi.NewRouter(),
		logger:  logger,
		handler: handler,
	}

	app.setupRoutes()
	logger.Info("NewApp: routes setup")

	return app, nil
}

// Run запускает приложение (этот метод не используется в текущей схеме main.go, где сервер создается и запускается в runServer)
// func (a *App) Run() error { ... }

// setupRoutes настраивает маршруты приложения
func (a *App) setupRoutes() {
	a.router.Use(a.handler.WithLogging)
	a.router.Use(a.handler.WithGzip)
	// a.router.Use(middleware.WithAuth) // Пока оставляем закомментированным для iteration1

	a.router.Post("/", a.handler.HandleCreateURL)
	a.router.Get("/{id}", a.handler.HandleRedirect)
	a.router.Post("/api/shorten", a.handler.HandleShortenURL)
	a.router.Post("/api/shorten/batch", a.handler.HandleShortenBatch)
	a.router.Get("ping", a.handler.HandlePing) // Убрал /api для соответствия тестам из других итераций
	a.router.Get("/api/user/urls", a.handler.HandleGetUserURLs)
}

// Configure настраивает все слои приложения (не используется в текущей схеме)
// func (a *App) Configure() error { ... }

// GetServer возвращает настроенный HTTP сервер
func (a *App) GetServer() *http.Server {
	return &http.Server{
		Addr:         a.config.ServerAddress, // Убедимся, что ServerAddress из cfg используется
		Handler:      a.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// Router возвращает HTTP роутер приложения (не используется напрямую в main)
// func (a *App) Router() http.Handler { ... }

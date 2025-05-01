package app

import (
	"net/http"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
)

// App представляет основное приложение
type App struct {
	config *config.Config
	router *chi.Mux
}

// New создает новый экземпляр приложения
func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
		router: chi.NewRouter(),
	}
}

// Configure настраивает все слои приложения
func (a *App) Configure() error {
	// Инициализация сервисов и обработчиков
	urlService, err := service.NewURLService(a.config)
	if err != nil {
		return err
	}
	handler := handler.NewHandler(urlService, a.config)

	// Подключаем middleware
	a.router.Use(middleware.GzipMiddleware)

	// Регистрация маршрутов
	a.router.Post("/", handler.HandleCreateURL)
	a.router.Post("/api/shorten", handler.HandleShortenURL)
	a.router.Get("/", handler.HandleRedirect)
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

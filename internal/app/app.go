// Package app содержит основную структуру приложения и логику инициализации.
// Предоставляет точку входа для запуска HTTP сервера с настроенными маршрутами и middleware.
package app

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// App представляет основное приложение сервиса сокращения URL.
// Инкапсулирует конфигурацию, HTTP роутер, логгер и обработчики запросов.
type App struct {
	config  *config.Config   // Конфигурация приложения
	router  *chi.Mux         // HTTP роутер для обработки запросов
	logger  *zap.Logger      // Логгер для записи событий приложения
	handler *handler.Handler // Обработчики HTTP запросов
}

// NewApp создает и инициализирует новый экземпляр приложения.
// Автоматически настраивает логгер, сервисный слой и обработчики запросов.
//
// Параметры:
//   - cfg: конфигурация приложения с настройками сервера и хранилища
//
// Возвращает указатель на App или ошибку при неудачной инициализации зависимостей.
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

// Run запускает HTTP или HTTPS сервер приложения в зависимости от конфигурации.
// Настраивает маршруты, создает HTTP сервер с таймаутами и начинает прослушивание запросов.
// Блокирующий вызов - выполняется до остановки сервера.
//
// Возвращает ошибку, если сервер не может быть запущен или произошла ошибка во время работы.
func (a *App) Run() error {
	a.setupRoutes()

	server := &http.Server{
		Addr:         a.config.ServerAddress,
		Handler:      a.router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	if a.config.IsHTTPSEnabled() {
		a.logger.Info("Starting HTTPS server",
			zap.String("address", a.config.ServerAddress),
			zap.String("cert", a.config.TLSCertFile),
			zap.String("key", a.config.TLSKeyFile))
		return server.ListenAndServeTLS(a.config.TLSCertFile, a.config.TLSKeyFile)
	}

	a.logger.Info("Starting HTTP server", zap.String("address", a.config.ServerAddress))
	return server.ListenAndServe()
}

// setupRoutes настраивает HTTP маршруты и middleware для приложения.
// Регистрирует все эндпоинты API и применяет глобальные middleware
// (логирование, сжатие, аутентификация).
func (a *App) setupRoutes() {
	// Middleware
	a.router.Use(a.handler.WithLogging)
	a.router.Use(a.handler.WithGzip)
	a.router.Use(a.handler.AuthMiddleware)

	// Routes
	a.router.Post("/", a.handler.HandleCreateURL)
	a.router.Get("/{id}", a.handler.HandleRedirect)
	a.router.Post("/api/shorten", a.handler.HandleShortenURL)
	a.router.Post("/api/shorten/batch", a.handler.HandleShortenBatch)
	a.router.Get("/ping", a.handler.HandlePing)
	a.router.Get("/api/user/urls", a.handler.HandleGetUserURLs)
	a.router.Delete("/api/user/urls", a.handler.HandleDeleteUserURLs)

	// Профилирование (доступно только в debug режиме)
	a.router.Mount("/debug/pprof", http.DefaultServeMux)
}

// Configure настраивает все слои приложения.
// Альтернативный метод инициализации, который создает сервисы и регистрирует маршруты.
// Похож на setupRoutes, но выполняет полную реинициализацию зависимостей.
//
// Возвращает ошибку при неудачной инициализации сервисного слоя.
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
	a.router.Use(handler.AuthMiddleware)

	// Регистрация маршрутов
	a.router.Post("/", handler.HandleCreateURL)
	a.router.Post("/api/shorten", handler.HandleShortenURL)
	a.router.Post("/api/shorten/batch", handler.HandleShortenBatch)
	a.router.Get("/{shortID}", handler.HandleRedirect)

	// Добавляем хендлер для проверки доступности БД
	a.router.Get("/ping", handler.HandlePing)
	a.router.Get("/api/user/urls", handler.HandleGetUserURLs)
	a.router.Delete("/api/user/urls", handler.HandleDeleteUserURLs)

	// Профилирование (доступно только в debug режиме)
	a.router.Mount("/debug/pprof", http.DefaultServeMux)

	return nil
}

// GetServer создает и возвращает настроенный HTTP сервер.
// Сервер настроен с оптимальными таймаутами для production использования.
// Использует текущий роутер приложения как обработчик запросов.
//
// Возвращает готовый к использованию http.Server с настроенными таймаутами.
func (a *App) GetServer() *http.Server {
	return &http.Server{
		Addr:         a.config.ServerAddress,
		Handler:      a.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

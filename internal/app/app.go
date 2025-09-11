// Package app содержит основную структуру приложения и логику инициализации.
// Предоставляет точку входа для запуска HTTP и gRPC серверов с настроенными маршрутами и middleware.
package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/grpc"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// App представляет основное приложение сервиса сокращения URL.
// Инкапсулирует конфигурацию, HTTP и gRPC серверы, логгер и обработчики запросов.
type App struct {
	config     *config.Config     // Конфигурация приложения
	router     *chi.Mux           // HTTP роутер для обработки запросов
	logger     *zap.Logger        // Логгер для записи событий приложения
	handler    *handler.Handler   // Обработчики HTTP запросов
	grpcServer *grpc.GRPCServer   // gRPC сервер
	service    service.URLService // URL сервис для бизнес-логики
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
	grpcServer := grpc.NewGRPCServer(cfg, logger)

	return &App{
		config:     cfg,
		router:     chi.NewRouter(),
		logger:     logger,
		handler:    handler,
		grpcServer: grpcServer,
		service:    service,
	}, nil
}

// Run запускает HTTP и/или gRPC серверы приложения в зависимости от конфигурации.
// Настраивает маршруты, создает серверы с таймаутами и начинает прослушивание запросов.
// Блокирующий вызов - выполняется до остановки серверов.
//
// Возвращает ошибку, если серверы не могут быть запущены или произошла ошибка во время работы.
func (a *App) Run() error {
	a.setupRoutes()

	var wg sync.WaitGroup
	var httpErr, grpcErr error

	// Запускаем HTTP сервер
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpErr = a.runHTTPServer()
	}()

	// Запускаем gRPC сервер если включен
	if a.config.EnableGRPC {
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpcErr = a.runGRPCServer()
		}()
	}

	// Ждем завершения всех серверов
	wg.Wait()

	// Возвращаем первую ошибку, если она есть
	if httpErr != nil {
		return fmt.Errorf("HTTP server error: %w", httpErr)
	}
	if grpcErr != nil {
		return fmt.Errorf("gRPC server error: %w", grpcErr)
	}

	return nil
}

// runHTTPServer запускает HTTP сервер
func (a *App) runHTTPServer() error {
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

// runGRPCServer запускает gRPC сервер
func (a *App) runGRPCServer() error {
	// Создаем gRPC обработчик
	grpcHandler := grpc.NewGRPCHandler(a.service, a.config, a.logger)

	// Регистрируем сервис
	a.grpcServer.RegisterService(grpcHandler)

	a.logger.Info("Starting gRPC server", zap.String("address", a.config.GRPCServerAddress))
	return a.grpcServer.Start()
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

	// Внутренние API с проверкой доверенной подсети
	a.router.With(middleware.TrustedSubnetMiddleware(a.config, a.logger)).Get("/api/internal/stats", a.handler.HandleGetStats)

	// gRPC HTTP адаптер (для тестирования gRPC через HTTP)
	if a.config.EnableGRPC {
		grpcAdapter := grpc.NewHTTPToGRPCAdapter(a.service, a.config, a.logger)
		a.router.Mount("/grpc", grpcAdapter)
	}

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

	// Сохраняем сервис в структуре для доступа при shutdown
	a.service = urlService

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

// GetService возвращает URL сервис для управления жизненным циклом
func (a *App) GetService() service.URLService {
	return a.service
}

// Close корректно закрывает приложение и освобождает ресурсы
func (a *App) Close() error {
	a.logger.Info("Closing application...")

	// Закрываем gRPC сервер если он запущен
	if a.grpcServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.grpcServer.Shutdown(ctx); err != nil {
			a.logger.Error("Error shutting down gRPC server", zap.Error(err))
		}
	}

	// Закрываем сервис
	if a.service != nil {
		if err := a.service.Close(); err != nil {
			a.logger.Error("Error closing service", zap.Error(err))
			return err
		}
	}

	a.logger.Info("Application closed successfully")
	return nil
}

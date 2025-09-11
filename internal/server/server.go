// Package server предоставляет общую функциональность для запуска HTTP и HTTPS серверов.
// Пакет инкапсулирует логику инициализации конфигурации, логгера и запуска серверов.
package server

import (
	"context"
	"log"
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"go.uber.org/zap"
)

// Starter интерфейс для запуска сервера
type Starter interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// HTTPServer представляет HTTP сервер с общей логикой запуска
type HTTPServer struct {
	server *http.Server
	config *config.Config
	logger *zap.Logger
}

// NewHTTPServer создает новый HTTP сервер
func NewHTTPServer(server *http.Server, cfg *config.Config, logger *zap.Logger) *HTTPServer {
	return &HTTPServer{
		server: server,
		config: cfg,
		logger: logger,
	}
}

// Start запускает HTTP или HTTPS сервер в зависимости от конфигурации
func (s *HTTPServer) Start() error {
	if s.config.IsHTTPSEnabled() {
		return s.startHTTPS()
	}
	return s.startHTTP()
}

// Shutdown корректно завершает работу сервера
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Используем встроенный метод graceful shutdown от http.Server
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Error during server shutdown", zap.Error(err))
		return err
	}

	s.logger.Info("Server shutdown completed")
	return nil
}

// startHTTPS запускает HTTPS сервер
func (s *HTTPServer) startHTTPS() error {
	s.logger.Info("Starting HTTPS server",
		zap.String("address", s.config.ServerAddress),
		zap.String("cert", s.config.TLSCertFile),
		zap.String("key", s.config.TLSKeyFile))

	return s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
}

// startHTTP запускает HTTP сервер
func (s *HTTPServer) startHTTP() error {
	s.logger.Info("Starting HTTP server", zap.String("address", s.config.ServerAddress))
	return s.server.ListenAndServe()
}

// InitLogger инициализирует production логгер с defer функцией для синхронизации
func InitLogger() (*zap.Logger, func()) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}

	cleanup := func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Error syncing logger: %v", err)
		}
	}

	return logger, cleanup
}

// InitConfig инициализирует конфигурацию приложения
func InitConfig(logger *zap.Logger) *config.Config {
	cfg, err := config.NewConfig()
	if err != nil {
		if logger != nil {
			logger.Fatal("Error loading config", zap.Error(err))
		} else {
			log.Fatalf("Error loading config: %v", err)
		}
	}
	return cfg
}

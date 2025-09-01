// Package grpc содержит gRPC обработчики для сервиса сокращения URL.
package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer представляет gRPC сервер
type GRPCServer struct {
	server *grpc.Server
	config *config.Config
	logger *zap.Logger
}

// NewGRPCServer создает новый gRPC сервер
func NewGRPCServer(cfg *config.Config, logger *zap.Logger) *GRPCServer {
	// Создаем gRPC сервер с опциями
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)

	return &GRPCServer{
		server: grpcServer,
		config: cfg,
		logger: logger,
	}
}

// RegisterService регистрирует gRPC сервис
func (s *GRPCServer) RegisterService(service URLServiceServer) {
	// В реальном gRPC здесь была бы регистрация через RegisterURLServiceServer
	// Но для нашей упрощенной версии мы просто сохраняем сервис
	s.logger.Info("gRPC service registered")
}

// Start запускает gRPC сервер
func (s *GRPCServer) Start() error {
	// Включаем reflection для отладки
	reflection.Register(s.server)

	// Создаем listener
	lis, err := net.Listen("tcp", s.config.GRPCServerAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", zap.String("address", s.config.GRPCServerAddress))

	// Запускаем сервер
	return s.server.Serve(lis)
}

// Shutdown корректно завершает работу gRPC сервера
func (s *GRPCServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down gRPC server...")

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	case <-done:
		s.logger.Info("gRPC server shutdown completed")
		return nil
	}
}

// GetServer возвращает внутренний gRPC сервер
func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}

// loggingInterceptor добавляет логирование для gRPC запросов
func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.Any("request", req))

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Error(err))
		} else {
			logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod))
		}

		return resp, err
	}
}

// HTTPToGRPCAdapter адаптирует HTTP запросы к gRPC
type HTTPToGRPCAdapter struct {
	grpcHandler *GRPCHandler
	logger      *zap.Logger
}

// NewHTTPToGRPCAdapter создает адаптер для HTTP к gRPC
func NewHTTPToGRPCAdapter(service service.URLService, cfg *config.Config, logger *zap.Logger) *HTTPToGRPCAdapter {
	return &HTTPToGRPCAdapter{
		grpcHandler: NewGRPCHandler(service, cfg, logger),
		logger:      logger,
	}
}

// ServeHTTP обрабатывает HTTP запросы, конвертируя их в gRPC вызовы
func (a *HTTPToGRPCAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Простая реализация для демонстрации
	// В реальном приложении здесь была бы более сложная логика маршрутизации

	switch r.URL.Path {
	case "/grpc/create":
		if r.Method == http.MethodPost {
			a.handleCreateURL(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/grpc/get":
		if r.Method == http.MethodGet {
			a.handleGetURL(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/grpc/ping":
		if r.Method == http.MethodGet {
			a.handlePing(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/grpc/stats":
		if r.Method == http.MethodGet {
			a.handleStats(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

// handleCreateURL обрабатывает создание URL через gRPC интерфейс
func (a *HTTPToGRPCAdapter) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	// В этой упрощенной версии мы используем query параметры
	// В реальном приложении здесь была бы десериализация тела запроса

	// Простая реализация - в реальном приложении здесь была бы десериализация
	originalURL := r.URL.Query().Get("url")
	if originalURL == "" {
		http.Error(w, "URL parameter required", http.StatusBadRequest)
		return
	}

	req := &CreateShortURLRequest{OriginalURL: originalURL}
	resp, err := a.grpcHandler.CreateShortURL(r.Context(), req)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(resp.StatusCode))
	if resp.ShortURL != "" {
		w.Write([]byte(fmt.Sprintf(`{"short_url": "%s"}`, resp.ShortURL)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, resp.ErrorMessage)))
	}
}

// handleGetURL обрабатывает получение URL через gRPC интерфейс
func (a *HTTPToGRPCAdapter) handleGetURL(w http.ResponseWriter, r *http.Request) {
	shortID := r.URL.Query().Get("id")
	if shortID == "" {
		http.Error(w, "ID parameter required", http.StatusBadRequest)
		return
	}

	req := &GetOriginalURLRequest{ShortID: shortID}
	resp, err := a.grpcHandler.GetOriginalURL(r.Context(), req)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(resp.StatusCode))
	if resp.OriginalURL != "" {
		w.Write([]byte(fmt.Sprintf(`{"original_url": "%s"}`, resp.OriginalURL)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, resp.ErrorMessage)))
	}
}

// handlePing обрабатывает ping через gRPC интерфейс
func (a *HTTPToGRPCAdapter) handlePing(w http.ResponseWriter, r *http.Request) {
	req := &PingRequest{}
	resp, err := a.grpcHandler.Ping(r.Context(), req)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(int(resp.StatusCode))
	if resp.ErrorMessage != "" {
		w.Write([]byte(resp.ErrorMessage))
	}
}

// handleStats обрабатывает статистику через gRPC интерфейс
func (a *HTTPToGRPCAdapter) handleStats(w http.ResponseWriter, r *http.Request) {
	req := &GetStatsRequest{}
	resp, err := a.grpcHandler.GetStats(r.Context(), req)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(resp.StatusCode))
	if resp.StatusCode == 200 {
		w.Write([]byte(fmt.Sprintf(`{"urls": %d, "users": %d}`, resp.UrlsCount, resp.UsersCount)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, resp.ErrorMessage)))
	}
}

package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// LoggerMiddleware создает middleware для логирования запросов и ответов
func LoggerMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Начало запроса
			start := time.Now()
			path := r.URL.Path
			method := r.Method

			// Создаем обертку для ResponseWriter, чтобы отслеживать статус и размер
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Продолжаем обработку запроса
			next.ServeHTTP(ww, r)

			// После обработки запроса
			latency := time.Since(start)
			status := ww.Status()
			size := ww.BytesWritten()

			// Логируем информацию
			logger.Info("Request processed",
				zap.String("path", path),
				zap.String("method", method),
				zap.Duration("latency", latency),
				zap.Int("status", status),
				zap.Int("size", size),
			)
		})
	}
}

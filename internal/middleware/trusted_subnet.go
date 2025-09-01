// Package middleware содержит middleware компоненты для HTTP сервера.
// Этот файл содержит middleware для проверки доверенной подсети.
package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"go.uber.org/zap"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента находится в доверенной подсети
func TrustedSubnetMiddleware(cfg *config.Config, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если доверенная подсеть не настроена, запрещаем доступ
			if cfg.TrustedSubnet == "" {
				logger.Warn("Access denied: trusted subnet not configured",
					zap.String("path", r.URL.Path),
					zap.String("remoteAddr", r.RemoteAddr))
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Получаем IP-адрес из заголовка X-Real-IP
			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				// Если заголовок отсутствует, используем RemoteAddr
				clientIP = extractIPFromRemoteAddr(r.RemoteAddr)
			}

			// Проверяем, что IP-адрес валидный
			ip := net.ParseIP(clientIP)
			if ip == nil {
				logger.Warn("Invalid client IP address",
					zap.String("clientIP", clientIP),
					zap.String("path", r.URL.Path))
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Парсим CIDR
			_, ipNet, err := net.ParseCIDR(cfg.TrustedSubnet)
			if err != nil {
				logger.Error("Invalid trusted subnet configuration",
					zap.String("trustedSubnet", cfg.TrustedSubnet),
					zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Проверяем, входит ли IP в доверенную подсеть
			if !ipNet.Contains(ip) {
				logger.Warn("Access denied: IP not in trusted subnet",
					zap.String("clientIP", clientIP),
					zap.String("trustedSubnet", cfg.TrustedSubnet),
					zap.String("path", r.URL.Path))
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			logger.Debug("Access granted: IP in trusted subnet",
				zap.String("clientIP", clientIP),
				zap.String("trustedSubnet", cfg.TrustedSubnet),
				zap.String("path", r.URL.Path))

			// Добавляем IP в контекст для возможного использования в обработчиках
			ctx := context.WithValue(r.Context(), "clientIP", clientIP)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractIPFromRemoteAddr извлекает IP-адрес из RemoteAddr
// RemoteAddr имеет формат "IP:PORT", например "192.168.1.1:12345"
func extractIPFromRemoteAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}

	// Убираем порт, если он есть
	if colonIndex := strings.LastIndex(remoteAddr, ":"); colonIndex != -1 {
		return remoteAddr[:colonIndex]
	}

	return remoteAddr
}

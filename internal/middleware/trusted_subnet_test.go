package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTrustedSubnetMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		trustedSubnet  string
		clientIP       string
		xRealIP        string
		expectedStatus int
	}{
		{
			name:           "no trusted subnet configured",
			trustedSubnet:  "",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "valid IP in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid IP in trusted subnet using X-Real-IP",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "127.0.0.1",
			xRealIP:        "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP not in trusted subnet",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "10.0.0.100",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "invalid IP address",
			trustedSubnet:  "192.168.1.0/24",
			clientIP:       "invalid-ip",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "invalid trusted subnet",
			trustedSubnet:  "invalid-subnet",
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем конфигурацию
			cfg := &config.Config{
				TrustedSubnet: tt.trustedSubnet,
			}

			// Создаем логгер
			logger, _ := zap.NewDevelopment()

			// Создаем middleware
			middleware := TrustedSubnetMiddleware(cfg, logger)

			// Создаем обработчик для тестирования
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Создаем запрос
			req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
			req.RemoteAddr = tt.clientIP + ":12345"
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			// Создаем response recorder
			w := httptest.NewRecorder()

			// Выполняем запрос
			handler.ServeHTTP(w, req)

			// Проверяем статус
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestExtractIPFromRemoteAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{
			name:       "valid IP with port",
			remoteAddr: "192.168.1.100:12345",
			expected:   "192.168.1.100",
		},
		{
			name:       "valid IP without port",
			remoteAddr: "192.168.1.100",
			expected:   "192.168.1.100",
		},
		{
			name:       "empty remote addr",
			remoteAddr: "",
			expected:   "",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:12345",
			expected:   "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIPFromRemoteAddr(tt.remoteAddr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

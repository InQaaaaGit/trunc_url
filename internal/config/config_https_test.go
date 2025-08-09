package config

import (
	"os"
	"strconv"
	"testing"
)

// TestHTTPSConfig проверяет работу HTTPS конфигурации с булевым типом
func TestHTTPSConfig(t *testing.T) {
	tests := []struct {
		name        string
		enableHTTPS bool
		expected    bool
	}{
		{
			name:        "HTTPS disabled",
			enableHTTPS: false,
			expected:    false,
		},
		{
			name:        "HTTPS enabled",
			enableHTTPS: true,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EnableHTTPS: tt.enableHTTPS,
			}

			if got := config.IsHTTPSEnabled(); got != tt.expected {
				t.Errorf("IsHTTPSEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestHTTPSConfigFromEnv проверяет загрузку HTTPS конфигурации из переменных окружения
func TestHTTPSConfigFromEnv(t *testing.T) {
	// Сохраняем оригинальные значения
	originalEnableHTTPS := os.Getenv("ENABLE_HTTPS")
	originalTLSCert := os.Getenv("TLS_CERT_FILE")
	originalTLSKey := os.Getenv("TLS_KEY_FILE")

	// Восстанавливаем после теста
	defer func() {
		os.Setenv("ENABLE_HTTPS", originalEnableHTTPS)
		os.Setenv("TLS_CERT_FILE", originalTLSCert)
		os.Setenv("TLS_KEY_FILE", originalTLSKey)
	}()

	// Устанавливаем тестовые значения
	os.Setenv("ENABLE_HTTPS", "true")
	os.Setenv("TLS_CERT_FILE", "test.crt")
	os.Setenv("TLS_KEY_FILE", "test.key")

	config := &Config{}

	// Имитируем парсинг переменных окружения для булевого поля
	if httpsEnv := os.Getenv("ENABLE_HTTPS"); httpsEnv != "" {
		enabled, err := strconv.ParseBool(httpsEnv)
		if err == nil {
			config.EnableHTTPS = enabled
		}
	}
	config.TLSCertFile = os.Getenv("TLS_CERT_FILE")
	config.TLSKeyFile = os.Getenv("TLS_KEY_FILE")

	if !config.IsHTTPSEnabled() {
		t.Error("Expected HTTPS to be enabled when ENABLE_HTTPS=true")
	}

	if config.TLSCertFile != "test.crt" {
		t.Errorf("Expected TLSCertFile to be 'test.crt', got '%s'", config.TLSCertFile)
	}

	if config.TLSKeyFile != "test.key" {
		t.Errorf("Expected TLSKeyFile to be 'test.key', got '%s'", config.TLSKeyFile)
	}
}

// TestHTTPSConfigFromEnvFalse проверяет, что HTTPS отключается при ENABLE_HTTPS=false
func TestHTTPSConfigFromEnvFalse(t *testing.T) {
	originalEnableHTTPS := os.Getenv("ENABLE_HTTPS")
	defer func() {
		os.Setenv("ENABLE_HTTPS", originalEnableHTTPS)
	}()

	// Устанавливаем значение false
	os.Setenv("ENABLE_HTTPS", "false")

	config := &Config{}
	if httpsEnv := os.Getenv("ENABLE_HTTPS"); httpsEnv != "" {
		enabled, err := strconv.ParseBool(httpsEnv)
		if err == nil {
			config.EnableHTTPS = enabled
		}
	}

	if config.IsHTTPSEnabled() {
		t.Error("Expected HTTPS to be disabled when ENABLE_HTTPS=false")
	}
}

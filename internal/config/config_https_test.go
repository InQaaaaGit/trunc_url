package config

import (
	"os"
	"testing"
)

func TestHTTPSConfig(t *testing.T) {
	tests := []struct {
		name        string
		enableHTTPS string
		expected    bool
	}{
		{
			name:        "HTTPS disabled empty string",
			enableHTTPS: "",
			expected:    false,
		},
		{
			name:        "HTTPS enabled with true",
			enableHTTPS: "true",
			expected:    true,
		},
		{
			name:        "HTTPS enabled with any value",
			enableHTTPS: "yes",
			expected:    true,
		},
		{
			name:        "HTTPS enabled with 1",
			enableHTTPS: "1",
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

	// Имитируем парсинг переменных окружения
	config.EnableHTTPS = os.Getenv("ENABLE_HTTPS")
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

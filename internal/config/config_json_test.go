package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/caarlos0/env/v6"
)

func TestLoadJSONConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedConfig *JSONConfig
		shouldError    bool
	}{
		{
			name:           "Empty filename",
			configContent:  "",
			expectedConfig: &JSONConfig{},
			shouldError:    false,
		},
		{
			name: "Valid JSON config",
			configContent: `{
				"server_address": "localhost:9090",
				"base_url": "https://example.com",
				"file_storage_path": "/tmp/urls.json",
				"database_dsn": "postgres://user:pass@localhost/db",
				"enable_https": true,
				"tls_cert_file": "custom.crt",
				"tls_key_file": "custom.key",
				"batch_delete_max_workers": 10
			}`,
			expectedConfig: &JSONConfig{
				ServerAddress:         stringPtr("localhost:9090"),
				BaseURL:               stringPtr("https://example.com"),
				FileStoragePath:       stringPtr("/tmp/urls.json"),
				DatabaseDSN:           stringPtr("postgres://user:pass@localhost/db"),
				EnableHTTPS:           boolPtr(true),
				TLSCertFile:           stringPtr("custom.crt"),
				TLSKeyFile:            stringPtr("custom.key"),
				BatchDeleteMaxWorkers: intPtr(10),
			},
			shouldError: false,
		},
		{
			name: "Partial JSON config",
			configContent: `{
				"server_address": ":3000",
				"enable_https": false
			}`,
			expectedConfig: &JSONConfig{
				ServerAddress: stringPtr(":3000"),
				EnableHTTPS:   boolPtr(false),
			},
			shouldError: false,
		},
		{
			name:           "Invalid JSON",
			configContent:  `{"invalid": json}`,
			expectedConfig: nil,
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filename string

			if tt.configContent != "" {
				// Создаем временный файл
				tmpfile, err := os.CreateTemp("", "test_config_*.json")
				if err != nil {
					t.Fatalf("Cannot create temp file: %v", err)
				}
				defer os.Remove(tmpfile.Name())

				if _, err := tmpfile.Write([]byte(tt.configContent)); err != nil {
					t.Fatalf("Cannot write to temp file: %v", err)
				}
				tmpfile.Close()

				filename = tmpfile.Name()
			}

			config, err := loadJSONConfig(filename)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Сравниваем результат
			if !compareJSONConfigs(config, tt.expectedConfig) {
				t.Errorf("Config mismatch.\nExpected: %+v\nGot: %+v", tt.expectedConfig, config)
			}
		})
	}
}

func TestApplyJSONConfig(t *testing.T) {
	// Создаем базовую конфигурацию с дефолтными значениями
	cfg := &Config{
		ServerAddress:                  ":8080",
		BaseURL:                        "http://localhost:8080",
		FileStoragePath:                "urls.json",
		DatabaseDSN:                    "",
		SecretKey:                      "your-secret-key",
		EnableHTTPS:                    "",
		TLSCertFile:                    "server.crt",
		TLSKeyFile:                     "server.key",
		BatchDeleteMaxWorkers:          3,
		BatchDeleteBatchSize:           5,
		BatchDeleteSequentialThreshold: 5,
	}

	jsonConfig := &JSONConfig{
		ServerAddress:                  stringPtr("localhost:9090"),
		BaseURL:                        stringPtr("https://example.com"),
		FileStoragePath:                stringPtr("/custom/path.json"),
		DatabaseDSN:                    stringPtr("postgres://localhost/mydb"),
		SecretKey:                      stringPtr("custom-secret"),
		EnableHTTPS:                    boolPtr(true),
		TLSCertFile:                    stringPtr("custom.crt"),
		TLSKeyFile:                     stringPtr("custom.key"),
		BatchDeleteMaxWorkers:          intPtr(10),
		BatchDeleteBatchSize:           intPtr(15),
		BatchDeleteSequentialThreshold: intPtr(20),
	}

	cfg.applyJSONConfig(jsonConfig)

	// Проверяем, что значения применились
	if cfg.ServerAddress != "localhost:9090" {
		t.Errorf("Expected ServerAddress 'localhost:9090', got '%s'", cfg.ServerAddress)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL 'https://example.com', got '%s'", cfg.BaseURL)
	}
	if cfg.FileStoragePath != "/custom/path.json" {
		t.Errorf("Expected FileStoragePath '/custom/path.json', got '%s'", cfg.FileStoragePath)
	}
	if cfg.DatabaseDSN != "postgres://localhost/mydb" {
		t.Errorf("Expected DatabaseDSN 'postgres://localhost/mydb', got '%s'", cfg.DatabaseDSN)
	}
	if cfg.SecretKey != "custom-secret" {
		t.Errorf("Expected SecretKey 'custom-secret', got '%s'", cfg.SecretKey)
	}
	if cfg.EnableHTTPS != "true" {
		t.Errorf("Expected EnableHTTPS 'true', got '%s'", cfg.EnableHTTPS)
	}
	if cfg.TLSCertFile != "custom.crt" {
		t.Errorf("Expected TLSCertFile 'custom.crt', got '%s'", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "custom.key" {
		t.Errorf("Expected TLSKeyFile 'custom.key', got '%s'", cfg.TLSKeyFile)
	}
	if cfg.BatchDeleteMaxWorkers != 10 {
		t.Errorf("Expected BatchDeleteMaxWorkers 10, got %d", cfg.BatchDeleteMaxWorkers)
	}
	if cfg.BatchDeleteBatchSize != 15 {
		t.Errorf("Expected BatchDeleteBatchSize 15, got %d", cfg.BatchDeleteBatchSize)
	}
	if cfg.BatchDeleteSequentialThreshold != 20 {
		t.Errorf("Expected BatchDeleteSequentialThreshold 20, got %d", cfg.BatchDeleteSequentialThreshold)
	}
}

func TestJSONConfigPriority(t *testing.T) {
	// Создаем временный JSON файл
	jsonContent := `{
		"server_address": "json:8080",
		"base_url": "http://json.com",
		"enable_https": true
	}`

	tmpfile, err := os.CreateTemp("", "test_priority_*.json")
	if err != nil {
		t.Fatalf("Cannot create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(jsonContent)); err != nil {
		t.Fatalf("Cannot write to temp file: %v", err)
	}
	tmpfile.Close()

	// Сохраняем оригинальные переменные окружения
	origServerAddr := os.Getenv("SERVER_ADDRESS")
	origBaseURL := os.Getenv("BASE_URL")
	defer func() {
		os.Setenv("SERVER_ADDRESS", origServerAddr)
		os.Setenv("BASE_URL", origBaseURL)
	}()

	// Устанавливаем переменную окружения
	os.Setenv("SERVER_ADDRESS", "env:8080")
	// BASE_URL намеренно не устанавливаем, чтобы проверить, что используется значение из JSON

	// Загружаем JSON конфигурацию
	jsonConfig, err := loadJSONConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	// Создаем базовую конфигурацию
	cfg := &Config{
		ServerAddress:                  ":8080",
		BaseURL:                        "http://localhost:8080",
		FileStoragePath:                "urls.json",
		DatabaseDSN:                    "",
		SecretKey:                      "your-secret-key",
		EnableHTTPS:                    "",
		TLSCertFile:                    "server.crt",
		TLSKeyFile:                     "server.key",
		BatchDeleteMaxWorkers:          3,
		BatchDeleteBatchSize:           5,
		BatchDeleteSequentialThreshold: 5,
	}

	// Применяем JSON конфигурацию (низший приоритет)
	cfg.applyJSONConfig(jsonConfig)

	// Применяем переменные окружения (средний приоритет)
	env.Parse(cfg)

	// Проверяем приоритеты:
	// ServerAddress должен быть из переменной окружения (env:8080)
	if cfg.ServerAddress != "env:8080" {
		t.Errorf("Expected ServerAddress from env 'env:8080', got '%s'", cfg.ServerAddress)
	}

	// BaseURL должен быть из JSON файла (http://json.com), так как переменная окружения не установлена
	if cfg.BaseURL != "http://json.com" {
		t.Errorf("Expected BaseURL from JSON 'http://json.com', got '%s'", cfg.BaseURL)
	}

	// EnableHTTPS должен быть из JSON файла (true)
	if cfg.EnableHTTPS != "true" {
		t.Errorf("Expected EnableHTTPS from JSON 'true', got '%s'", cfg.EnableHTTPS)
	}
}

func TestJSONConfigFileNotFound(t *testing.T) {
	// Тестируем случай, когда файл не существует
	config, err := loadJSONConfig("/nonexistent/config.json")
	if err != nil {
		t.Errorf("Expected no error for nonexistent file, got: %v", err)
	}
	if config == nil {
		t.Error("Expected empty config, got nil")
	}
}

// Вспомогательные функции

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func compareJSONConfigs(a, b *JSONConfig) bool {
	// Простое сравнение через JSON marshal/unmarshal
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

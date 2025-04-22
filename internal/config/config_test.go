package config

import (
	"flag"
	"os"
	"testing"
)

func TestConfigPriority(t *testing.T) {
	// Сохраняем оригинальные значения переменных окружения
	originalServerAddress := os.Getenv("SERVER_ADDRESS")
	originalBaseURL := os.Getenv("BASE_URL")
	defer func() {
		// Восстанавливаем оригинальные значения после теста
		if originalServerAddress != "" {
			os.Setenv("SERVER_ADDRESS", originalServerAddress)
		} else {
			os.Unsetenv("SERVER_ADDRESS")
		}
		if originalBaseURL != "" {
			os.Setenv("BASE_URL", originalBaseURL)
		} else {
			os.Unsetenv("BASE_URL")
		}
	}()

	// Сохраняем оригинальные аргументы командной строки
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name           string
		envServerAddr  string
		envBaseURL     string
		args           []string
		wantServerAddr string
		wantBaseURL    string
	}{
		{
			name:           "Default values",
			envServerAddr:  "",
			envBaseURL:     "",
			args:           []string{"cmd"},
			wantServerAddr: ":8080",
			wantBaseURL:    "http://localhost:8080",
		},
		{
			name:           "Environment variables override defaults",
			envServerAddr:  ":9090",
			envBaseURL:     "http://example.com",
			args:           []string{"cmd"},
			wantServerAddr: ":9090",
			wantBaseURL:    "http://example.com",
		},
		{
			name:           "Command line flags override defaults",
			envServerAddr:  "",
			envBaseURL:     "",
			args:           []string{"cmd", "-a", ":7070", "-b", "http://test.com"},
			wantServerAddr: ":7070",
			wantBaseURL:    "http://test.com",
		},
		{
			name:           "Environment variables override command line flags",
			envServerAddr:  ":9090",
			envBaseURL:     "http://example.com",
			args:           []string{"cmd", "-a", ":7070", "-b", "http://test.com"},
			wantServerAddr: ":9090",
			wantBaseURL:    "http://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			if tt.envServerAddr != "" {
				os.Setenv("SERVER_ADDRESS", tt.envServerAddr)
			} else {
				os.Unsetenv("SERVER_ADDRESS")
			}
			if tt.envBaseURL != "" {
				os.Setenv("BASE_URL", tt.envBaseURL)
			} else {
				os.Unsetenv("BASE_URL")
			}

			// Устанавливаем аргументы командной строки
			os.Args = tt.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Получаем конфигурацию
			cfg, err := NewConfig()
			if err != nil {
				t.Fatalf("NewConfig() error = %v", err)
			}

			// Проверяем значения
			if cfg.ServerAddress != tt.wantServerAddr {
				t.Errorf("ServerAddress = %v, want %v", cfg.ServerAddress, tt.wantServerAddr)
			}
			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.wantBaseURL)
			}
		})
	}
}

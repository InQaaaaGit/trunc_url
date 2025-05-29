package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// Config хранит конфигурацию приложения.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`    // Адрес для запуска HTTP-сервера
	BaseURL         string `env:"BASE_URL"`          // Базовый адрес для сокращенных URL
	FileStoragePath string `env:"FILE_STORAGE_PATH"` // Путь к файлу для хранения URL
	DatabaseDSN     string `env:"DATABASE_DSN"`      // Строка подключения к базе данных PostgreSQL
}

// NewConfig инициализирует конфигурацию, читая флаги и переменные окружения.
func NewConfig() (*Config, error) {
	cfg := &Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "urls.json",
		DatabaseDSN:     "",
	}

	// Определяем флаги
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый URL для сокращенных ссылок")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь к файлу для хранения URL")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных PostgreSQL")

	// Парсим флаги
	flag.Parse()

	// Парсим переменные окружения (имеет наивысший приоритет)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

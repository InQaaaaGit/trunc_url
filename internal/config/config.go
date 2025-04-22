package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// Config хранит конфигурацию приложения.
type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS"` // Адрес для запуска HTTP-сервера
	BaseURL       string `env:"BASE_URL"`       // Базовый адрес для сокращенных URL
}

// NewConfig инициализирует конфигурацию, читая флаги и переменные окружения.
func NewConfig() (*Config, error) {
	cfg := &Config{
		ServerAddress: ":8080",                 // Значение по умолчанию
		BaseURL:       "http://localhost:8080", // Значение по умолчанию
	}

	// 1. Определение флагов командной строки
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "Адрес запуска HTTP-сервера (env: SERVER_ADDRESS)")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Базовый адрес результирующего сокращённого URL (env: BASE_URL)")

	// 2. Парсинг флагов командной строки
	flag.Parse()

	// 3. Парсинг переменных окружения (имеет наивысший приоритет)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

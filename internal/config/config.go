package config

import (
	"flag"
	"os" // Импортируем пакет os для работы с переменными окружения
)

// Config хранит конфигурацию приложения.
type Config struct {
	ServerAddress string // Адрес для запуска HTTP-сервера (-a или SERVER_ADDRESS)
	BaseURL       string // Базовый адрес для сокращенных URL (-b или BASE_URL)
}

// NewConfig инициализирует конфигурацию, читая флаги и переменные окружения.
func NewConfig() *Config {
	cfg := &Config{}

	// 1. Определение флагов командной строки (значения по умолчанию)
	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "Адрес запуска HTTP-сервера (env: SERVER_ADDRESS)")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL (env: BASE_URL)")

	// 2. Парсинг флагов командной строки
	// Теперь cfg.ServerAddress и cfg.BaseURL содержат либо значение флага, либо значение по умолчанию
	flag.Parse()

	// 3. Проверка переменных окружения и перезапись значений при необходимости
	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		cfg.ServerAddress = envServerAddress
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}

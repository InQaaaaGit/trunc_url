package config

import "flag"

// Config хранит конфигурацию приложения.
type Config struct {
	ServerAddress string // Адрес для запуска HTTP-сервера (-a)
	BaseURL       string // Базовый адрес для сокращенных URL (-b)
}

// NewConfig инициализирует конфигурацию, парсит флаги командной строки.
func NewConfig() *Config {
	cfg := &Config{}

	// Определение флагов командной строки
	// Первый аргумент - имя флага
	// Второй - значение по умолчанию
	// Третий - описание флага
	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "Адрес запуска HTTP-сервера (формат: хост:порт)")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL")

	// Парсинг флагов
	flag.Parse()

	return cfg
}

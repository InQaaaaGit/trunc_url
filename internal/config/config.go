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
	SecretKey       string `env:"SECRET_KEY"`        // Секретный ключ для подписи кук

	// Параметры для batch deletion
	BatchDeleteMaxWorkers          int `env:"BATCH_DELETE_MAX_WORKERS"`          // Максимальное количество воркеров для параллельного удаления
	BatchDeleteBatchSize           int `env:"BATCH_DELETE_BATCH_SIZE"`           // Размер батча для обработки URL
	BatchDeleteSequentialThreshold int `env:"BATCH_DELETE_SEQUENTIAL_THRESHOLD"` // Порог для переключения на последовательное удаление
}

// NewConfig инициализирует конфигурацию, читая флаги и переменные окружения.
func NewConfig() (*Config, error) {
	cfg := &Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "urls.json",
		DatabaseDSN:     "",
		SecretKey:       "your-secret-key", // Значение по умолчанию, лучше изменить

		// Значения по умолчанию для batch deletion
		BatchDeleteMaxWorkers:          3,
		BatchDeleteBatchSize:           5,
		BatchDeleteSequentialThreshold: 5,
	}

	// Определяем флаги
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый URL для сокращенных ссылок")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь к файлу для хранения URL")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных PostgreSQL")
	flag.StringVar(&cfg.SecretKey, "s", cfg.SecretKey, "секретный ключ для подписи кук")

	// Флаги для настройки batch deletion
	flag.IntVar(&cfg.BatchDeleteMaxWorkers, "batch-max-workers", cfg.BatchDeleteMaxWorkers, "максимальное количество воркеров для параллельного удаления URL")
	flag.IntVar(&cfg.BatchDeleteBatchSize, "batch-size", cfg.BatchDeleteBatchSize, "размер батча для обработки URL")
	flag.IntVar(&cfg.BatchDeleteSequentialThreshold, "batch-sequential-threshold", cfg.BatchDeleteSequentialThreshold, "порог для переключения на последовательное удаление URL")

	// Парсим флаги
	flag.Parse()

	// Парсим переменные окружения (имеет наивысший приоритет)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

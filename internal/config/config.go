// Package config предоставляет функциональность для загрузки и управления конфигурацией приложения.
// Конфигурация может загружаться из флагов командной строки, переменных окружения и JSON файла.
// Приоритет: флаги > переменные окружения > JSON файл.
package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

// Config содержит все настройки конфигурации приложения.
// Поддерживает загрузку из флагов командной строки, переменных окружения и JSON файла.
// Приоритет конфигурации: флаги > переменные окружения > JSON файл.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`    // Адрес для запуска HTTP-сервера (например, ":8080")
	BaseURL         string `env:"BASE_URL"`          // Базовый адрес для сокращенных URL (например, "http://localhost:8080")
	FileStoragePath string `env:"FILE_STORAGE_PATH"` // Путь к файлу для хранения URL (например, "urls.json")
	DatabaseDSN     string `env:"DATABASE_DSN"`      // Строка подключения к базе данных PostgreSQL
	SecretKey       string `env:"SECRET_KEY"`        // Секретный ключ для подписи аутентификационных кук

	// HTTPS настройки
	EnableHTTPS string `env:"ENABLE_HTTPS"`  // Включить HTTPS сервер
	TLSCertFile string `env:"TLS_CERT_FILE"` // Путь к файлу сертификата TLS
	TLSKeyFile  string `env:"TLS_KEY_FILE"`  // Путь к файлу приватного ключа TLS

	// Параметры для batch deletion
	BatchDeleteMaxWorkers          int `env:"BATCH_DELETE_MAX_WORKERS"`          // Максимальное количество воркеров для параллельного удаления
	BatchDeleteBatchSize           int `env:"BATCH_DELETE_BATCH_SIZE"`           // Размер батча для обработки URL
	BatchDeleteSequentialThreshold int `env:"BATCH_DELETE_SEQUENTIAL_THRESHOLD"` // Порог для переключения на последовательное удаление

	// Конфигурационный файл
	ConfigFile string `env:"CONFIG"` // Путь к JSON файлу конфигурации
}

// JSONConfig представляет структуру JSON файла конфигурации.
// Все поля являются указателями для различения "не установлено" от "пустое значение".
type JSONConfig struct {
	ServerAddress   *string `json:"server_address,omitempty"`    // Адрес для запуска HTTP-сервера
	BaseURL         *string `json:"base_url,omitempty"`          // Базовый адрес для сокращенных URL
	FileStoragePath *string `json:"file_storage_path,omitempty"` // Путь к файлу для хранения URL
	DatabaseDSN     *string `json:"database_dsn,omitempty"`      // Строка подключения к базе данных PostgreSQL
	SecretKey       *string `json:"secret_key,omitempty"`        // Секретный ключ для подписи аутентификационных кук

	// HTTPS настройки
	EnableHTTPS *bool   `json:"enable_https,omitempty"`  // Включить HTTPS сервер
	TLSCertFile *string `json:"tls_cert_file,omitempty"` // Путь к файлу сертификата TLS
	TLSKeyFile  *string `json:"tls_key_file,omitempty"`  // Путь к файлу приватного ключа TLS

	// Параметры для batch deletion
	BatchDeleteMaxWorkers          *int `json:"batch_delete_max_workers,omitempty"`          // Максимальное количество воркеров для параллельного удаления
	BatchDeleteBatchSize           *int `json:"batch_delete_batch_size,omitempty"`           // Размер батча для обработки URL
	BatchDeleteSequentialThreshold *int `json:"batch_delete_sequential_threshold,omitempty"` // Порог для переключения на последовательное удаление
}

// loadJSONConfig загружает конфигурацию из JSON файла.
// Возвращает заполненную JSONConfig или ошибку.
func loadJSONConfig(filename string) (*JSONConfig, error) {
	if filename == "" {
		return &JSONConfig{}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует - это не ошибка, просто используем пустую конфигурацию
			return &JSONConfig{}, nil
		}
		return nil, err
	}

	var jsonConfig JSONConfig
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return nil, err
	}

	return &jsonConfig, nil
}

// applyJSONConfig применяет значения из JSON конфигурации к Config,
// но только если соответствующие поля еще не установлены.
func (c *Config) applyJSONConfig(jsonConfig *JSONConfig) {
	if c.ServerAddress == ":8080" && jsonConfig.ServerAddress != nil {
		c.ServerAddress = *jsonConfig.ServerAddress
	}
	if c.BaseURL == "http://localhost:8080" && jsonConfig.BaseURL != nil {
		c.BaseURL = *jsonConfig.BaseURL
	}
	if c.FileStoragePath == "urls.json" && jsonConfig.FileStoragePath != nil {
		c.FileStoragePath = *jsonConfig.FileStoragePath
	}
	if c.DatabaseDSN == "" && jsonConfig.DatabaseDSN != nil {
		c.DatabaseDSN = *jsonConfig.DatabaseDSN
	}
	if c.SecretKey == "your-secret-key" && jsonConfig.SecretKey != nil {
		c.SecretKey = *jsonConfig.SecretKey
	}

	// HTTPS настройки
	if c.EnableHTTPS == "" && jsonConfig.EnableHTTPS != nil {
		if *jsonConfig.EnableHTTPS {
			c.EnableHTTPS = "true"
		}
	}
	if c.TLSCertFile == "server.crt" && jsonConfig.TLSCertFile != nil {
		c.TLSCertFile = *jsonConfig.TLSCertFile
	}
	if c.TLSKeyFile == "server.key" && jsonConfig.TLSKeyFile != nil {
		c.TLSKeyFile = *jsonConfig.TLSKeyFile
	}

	// Batch deletion настройки
	if c.BatchDeleteMaxWorkers == 3 && jsonConfig.BatchDeleteMaxWorkers != nil {
		c.BatchDeleteMaxWorkers = *jsonConfig.BatchDeleteMaxWorkers
	}
	if c.BatchDeleteBatchSize == 5 && jsonConfig.BatchDeleteBatchSize != nil {
		c.BatchDeleteBatchSize = *jsonConfig.BatchDeleteBatchSize
	}
	if c.BatchDeleteSequentialThreshold == 5 && jsonConfig.BatchDeleteSequentialThreshold != nil {
		c.BatchDeleteSequentialThreshold = *jsonConfig.BatchDeleteSequentialThreshold
	}
}

// NewConfig создает и инициализирует новую конфигурацию приложения.
// Применяет конфигурацию в следующем порядке приоритета:
// 1. Значения по умолчанию
// 2. JSON файл конфигурации (если указан)
// 3. Переменные окружения
// 4. Флаги командной строки (наивысший приоритет)
//
// Возвращает указатель на Config или ошибку при неудачной инициализации.
func NewConfig() (*Config, error) {
	cfg := &Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "urls.json",
		DatabaseDSN:     "",
		SecretKey:       "your-secret-key", // Значение по умолчанию, лучше изменить

		// HTTPS настройки по умолчанию
		EnableHTTPS: "",
		TLSCertFile: "server.crt",
		TLSKeyFile:  "server.key",

		// Значения по умолчанию для batch deletion
		BatchDeleteMaxWorkers:          3,
		BatchDeleteBatchSize:           5,
		BatchDeleteSequentialThreshold: 5,

		// Конфигурационный файл
		ConfigFile: "",
	}

	// Определяем флаги
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый URL для сокращенных ссылок")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь к файлу для хранения URL")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных PostgreSQL")
	flag.StringVar(&cfg.SecretKey, "secret-key", cfg.SecretKey, "секретный ключ для подписи кук")

	// HTTPS флаги
	flag.StringVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "включить HTTPS сервер")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "путь к файлу TLS сертификата")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "путь к файлу TLS приватного ключа")

	// Флаги для настройки batch deletion
	flag.IntVar(&cfg.BatchDeleteMaxWorkers, "batch-max-workers", cfg.BatchDeleteMaxWorkers, "максимальное количество воркеров для параллельного удаления URL")
	flag.IntVar(&cfg.BatchDeleteBatchSize, "batch-size", cfg.BatchDeleteBatchSize, "размер батча для обработки URL")
	flag.IntVar(&cfg.BatchDeleteSequentialThreshold, "batch-sequential-threshold", cfg.BatchDeleteSequentialThreshold, "порог для переключения на последовательное удаление URL")

	// Флаг для JSON конфигурации
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "путь к JSON файлу конфигурации")
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "путь к JSON файлу конфигурации")

	// Сначала парсим флаги чтобы получить путь к конфигурационному файлу
	flag.Parse()

	// Проверяем переменную окружения CONFIG для пути к файлу конфигурации
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" && cfg.ConfigFile == "" {
		cfg.ConfigFile = envConfigFile
	}

	// Загружаем JSON конфигурацию (если указана)
	jsonConfig, err := loadJSONConfig(cfg.ConfigFile)
	if err != nil {
		return nil, err
	}

	// Применяем JSON конфигурацию (самый низкий приоритет)
	cfg.applyJSONConfig(jsonConfig)

	// Парсим переменные окружения (средний приоритет)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Флаги уже применены flag.Parse() выше (наивысший приоритет)

	return cfg, nil
}

// IsHTTPSEnabled проверяет, включен ли HTTPS режим.
// HTTPS включен если флаг -s передан (любое непустое значение) или установлена переменная окружения ENABLE_HTTPS.
func (c *Config) IsHTTPSEnabled() bool {
	return c.EnableHTTPS != ""
}

package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewConfigDefaults проверяет создание конфигурации с значениями по умолчанию
func TestNewConfigDefaults(t *testing.T) {
	// Reset flags for clean test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Clear environment variables
	os.Clearenv()

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "urls.json", cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "your-secret-key", cfg.SecretKey)
	assert.Equal(t, 3, cfg.BatchDeleteMaxWorkers)
	assert.Equal(t, 5, cfg.BatchDeleteBatchSize)
	assert.Equal(t, 5, cfg.BatchDeleteSequentialThreshold)
}

// TestNewConfigEnvironmentVariables проверяет загрузку конфигурации из переменных окружения
func TestNewConfigEnvironmentVariables(t *testing.T) {
	// Reset flags for clean test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set environment variables
	os.Setenv("SERVER_ADDRESS", ":9090")
	os.Setenv("BASE_URL", "https://example.com")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/urls.json")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")
	os.Setenv("SECRET_KEY", "env-secret-key")
	os.Setenv("BATCH_DELETE_MAX_WORKERS", "10")
	os.Setenv("BATCH_DELETE_BATCH_SIZE", "20")
	os.Setenv("BATCH_DELETE_SEQUENTIAL_THRESHOLD", "15")

	defer func() {
		os.Clearenv()
	}()

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.ServerAddress)
	assert.Equal(t, "https://example.com", cfg.BaseURL)
	assert.Equal(t, "/tmp/urls.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "env-secret-key", cfg.SecretKey)
	assert.Equal(t, 10, cfg.BatchDeleteMaxWorkers)
	assert.Equal(t, 20, cfg.BatchDeleteBatchSize)
	assert.Equal(t, 15, cfg.BatchDeleteSequentialThreshold)
}

// TestNewConfigCommandLineFlags проверяет загрузку конфигурации из флагов командной строки
func TestNewConfigCommandLineFlags(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Reset flags and environment
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Clearenv()

	// Set command line arguments
	os.Args = []string{
		"test",
		"-a", ":7070",
		"-b", "http://test.local",
		"-f", "/test/urls.json",
		"-d", "postgres://test",
		"-s", "flag-secret",
		"-batch-max-workers", "8",
		"-batch-size", "12",
		"-batch-sequential-threshold", "10",
	}

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, ":7070", cfg.ServerAddress)
	assert.Equal(t, "http://test.local", cfg.BaseURL)
	assert.Equal(t, "/test/urls.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://test", cfg.DatabaseDSN)
	assert.Equal(t, "flag-secret", cfg.SecretKey)
	assert.Equal(t, 8, cfg.BatchDeleteMaxWorkers)
	assert.Equal(t, 12, cfg.BatchDeleteBatchSize)
	assert.Equal(t, 10, cfg.BatchDeleteSequentialThreshold)
}

// TestNewConfigEnvironmentOverridesFlags проверяет приоритет переменных окружения над флагами
func TestNewConfigEnvironmentOverridesFlags(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
	}()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set command line arguments
	os.Args = []string{
		"test",
		"-a", ":7070",
		"-b", "http://flag.local",
	}

	// Set environment variables (should override flags)
	os.Setenv("SERVER_ADDRESS", ":6060")
	os.Setenv("BASE_URL", "http://env.local")

	cfg, err := NewConfig()
	require.NoError(t, err)

	// Environment should override flags
	assert.Equal(t, ":6060", cfg.ServerAddress)
	assert.Equal(t, "http://env.local", cfg.BaseURL)
}

// TestConfigAllFields проверяет наличие всех необходимых полей в структуре Config
func TestConfigAllFields(t *testing.T) {
	// Test that all expected fields exist in Config struct
	cfg := &Config{}

	// Basic fields
	assert.IsType(t, "", cfg.ServerAddress)
	assert.IsType(t, "", cfg.BaseURL)
	assert.IsType(t, "", cfg.FileStoragePath)
	assert.IsType(t, "", cfg.DatabaseDSN)
	assert.IsType(t, "", cfg.SecretKey)

	// Batch deletion fields
	assert.IsType(t, 0, cfg.BatchDeleteMaxWorkers)
	assert.IsType(t, 0, cfg.BatchDeleteBatchSize)
	assert.IsType(t, 0, cfg.BatchDeleteSequentialThreshold)
}

package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultInfo проверяет создание информации о сборке по умолчанию
func TestDefaultInfo(t *testing.T) {
	info := DefaultInfo()

	assert.Equal(t, "N/A", info.Version)
	assert.Equal(t, "N/A", info.Date)
	assert.Equal(t, "N/A", info.Commit)
}

// TestNewInfo проверяет создание информации о сборке с заданными параметрами
func TestNewInfo(t *testing.T) {
	version := "v1.0.0"
	date := "2024-01-01"
	commit := "abc123"

	info := NewInfo(version, date, commit)

	assert.Equal(t, version, info.Version)
	assert.Equal(t, date, info.Date)
	assert.Equal(t, commit, info.Commit)
}

// TestString проверяет строковое представление информации о сборке
func TestString(t *testing.T) {
	info := NewInfo("v1.0.0", "2024-01-01", "abc123")
	str := info.String()

	assert.Contains(t, str, "v1.0.0")
	assert.Contains(t, str, "2024-01-01")
	assert.Contains(t, str, "abc123")
	assert.Contains(t, str, "Version:")
	assert.Contains(t, str, "Date:")
	assert.Contains(t, str, "Commit:")
}

// TestPrint проверяет вывод информации о сборке (косвенно)
func TestPrint(t *testing.T) {
	info := NewInfo("v1.0.0", "2024-01-01", "abc123")

	// Проверяем, что метод не паникует
	assert.NotPanics(t, func() {
		info.Print()
	})
}

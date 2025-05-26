package storage

import (
	"context"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTestContext создает контекст с тестовым userID
func createTestContext() context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, middleware.UserIDKey, "test_user")
}

func TestNewMemoryStorage(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	assert.NotNil(t, storage)
	assert.NotNil(t, storage.urls)
	assert.NotNil(t, storage.users)
}

func TestMemoryStorage_SaveURLAndGetOriginalURL(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := createTestContext()

	// Тест сохранения и получения URL
	err = storage.SaveURL(ctx, "short1", "https://example.com")
	require.NoError(t, err)

	url, err := storage.GetOriginalURL(ctx, "short1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	// Тест получения несуществующего URL
	_, err = storage.GetOriginalURL(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestMemoryStorage_GetShortURL(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := createTestContext()

	// Сохраняем URL
	err = storage.SaveURL(ctx, "short1", "https://example.com")
	require.NoError(t, err)

	// Получаем короткий URL по оригинальному
	shortURL, err := storage.GetShortURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "short1", shortURL)

	// Тест с несуществующим URL
	_, err = storage.GetShortURL(ctx, "https://nonexistent.com")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestMemoryStorage_GetUserURLs(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := createTestContext()

	// Сохраняем URL для пользователя
	err = storage.SaveURL(ctx, "short1", "https://example.com")
	require.NoError(t, err)

	// Получаем URL пользователя
	urls, err := storage.GetUserURLs(ctx, "test_user")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "short1", urls[0].ShortURL)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)

	// Тест с несуществующим пользователем
	urls, err = storage.GetUserURLs(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestMemoryStorage_CheckConnection(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := createTestContext()

	// Проверяем соединение (для in-memory всегда должно быть OK)
	err = storage.CheckConnection(ctx)
	assert.NoError(t, err)
}

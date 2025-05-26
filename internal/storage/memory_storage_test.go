package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewMemoryStorage(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	assert.NotNil(t, storage)
	assert.NotNil(t, storage.urls)
	assert.NotNil(t, storage.userURLs)
}

func TestMemoryStorage_SaveAndGet(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := context.Background()

	// Тест сохранения и получения URL
	err = storage.Save(ctx, "short1", "https://example.com")
	require.NoError(t, err)

	url, err := storage.Get(ctx, "short1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	// Тест получения несуществующего URL
	_, err = storage.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestMemoryStorage_SaveUserURL(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := context.Background()

	// Тест сохранения URL пользователя
	err = storage.SaveUserURL(ctx, "user1", "short1", "https://example.com")
	require.NoError(t, err)

	// Проверяем, что URL сохранился
	url, err := storage.Get(ctx, "short1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	// Проверяем, что URL привязан к пользователю
	urls, err := storage.GetUserURLs(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "short1", urls[0].ShortURL)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)
}

func TestMemoryStorage_GetShortURLByOriginal(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := context.Background()

	// Сохраняем URL
	err = storage.Save(ctx, "short1", "https://example.com")
	require.NoError(t, err)

	// Получаем короткий URL по оригинальному
	shortURL, err := storage.GetShortURLByOriginal(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "short1", shortURL)

	// Тест с несуществующим URL
	_, err = storage.GetShortURLByOriginal(ctx, "https://nonexistent.com")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestMemoryStorage_SaveBatch(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := context.Background()

	batch := []BatchEntry{
		{
			ShortURL:    "short1",
			OriginalURL: "https://example1.com",
			UserID:      "user1",
		},
		{
			ShortURL:    "short2",
			OriginalURL: "https://example2.com",
			UserID:      "user1",
		},
	}

	err = storage.SaveBatch(ctx, batch)
	require.NoError(t, err)

	// Проверяем, что все URL сохранились
	url1, err := storage.Get(ctx, "short1")
	require.NoError(t, err)
	assert.Equal(t, "https://example1.com", url1)

	url2, err := storage.Get(ctx, "short2")
	require.NoError(t, err)
	assert.Equal(t, "https://example2.com", url2)

	// Проверяем, что все URL привязаны к пользователю
	urls, err := storage.GetUserURLs(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, urls, 2)
}

func TestMemoryStorage_GetUserURLs(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	storage := NewMemoryStorage(logger)
	ctx := context.Background()

	// Сохраняем URL для пользователя
	err = storage.SaveUserURL(ctx, "user1", "short1", "https://example.com")
	require.NoError(t, err)

	// Получаем URL пользователя
	urls, err := storage.GetUserURLs(ctx, "user1")
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
	ctx := context.Background()

	// Проверяем соединение (для in-memory всегда должно быть OK)
	err = storage.CheckConnection(ctx)
	assert.NoError(t, err)
}

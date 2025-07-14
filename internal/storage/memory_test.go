package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMemoryStorage_Save(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Test successful save
	err := storage.Save(ctx, "abc123", "https://example.com", "user1")
	assert.NoError(t, err)

	// Test retrieving saved URL
	originalURL, err := storage.Get(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", originalURL)
}

func TestMemoryStorage_Get(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Test getting non-existent URL
	_, err := storage.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrURLNotFound)

	// Test getting deleted URL
	_ = storage.Save(ctx, "deleted123", "https://example.com", "user1")
	_ = storage.BatchDelete(ctx, []string{"deleted123"}, "user1")

	_, err = storage.Get(ctx, "deleted123")
	assert.ErrorIs(t, err, ErrURLDeleted)
}

func TestMemoryStorage_GetShortURLByOriginal(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Save URL first
	err := storage.Save(ctx, "abc123", "https://example.com", "user1")
	assert.NoError(t, err)

	// Test getting short URL by original
	shortURL, err := storage.GetShortURLByOriginal(ctx, "https://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", shortURL)

	// Test getting non-existent original URL
	_, err = storage.GetShortURLByOriginal(ctx, "https://nonexistent.com")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestMemoryStorage_SaveBatch(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	batch := []BatchEntry{
		{ShortURL: "abc1", OriginalURL: "https://example1.com"},
		{ShortURL: "abc2", OriginalURL: "https://example2.com"},
	}

	// Test successful batch save
	err := storage.SaveBatch(ctx, batch)
	assert.NoError(t, err)

	// Verify saved URLs
	url1, err := storage.Get(ctx, "abc1")
	assert.NoError(t, err)
	assert.Equal(t, "https://example1.com", url1)

	url2, err := storage.Get(ctx, "abc2")
	assert.NoError(t, err)
	assert.Equal(t, "https://example2.com", url2)
}

func TestMemoryStorage_GetUserURLs(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Save URLs for different users
	_ = storage.Save(ctx, "abc1", "https://example1.com", "user1")
	_ = storage.Save(ctx, "abc2", "https://example2.com", "user1")
	_ = storage.Save(ctx, "abc3", "https://example3.com", "user2")

	// Get URLs for user1
	urls, err := storage.GetUserURLs(ctx, "user1")
	assert.NoError(t, err)
	assert.Len(t, urls, 2)

	// Check that correct URLs are returned
	urlMap := make(map[string]string)
	for _, url := range urls {
		urlMap[url.ShortURL] = url.OriginalURL
	}
	assert.Equal(t, "https://example1.com", urlMap["abc1"])
	assert.Equal(t, "https://example2.com", urlMap["abc2"])

	// Get URLs for user2
	urls, err = storage.GetUserURLs(ctx, "user2")
	assert.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "abc3", urls[0].ShortURL)
	assert.Equal(t, "https://example3.com", urls[0].OriginalURL)

	// Get URLs for non-existent user
	urls, err = storage.GetUserURLs(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Len(t, urls, 0)
}

func TestMemoryStorage_BatchDelete(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Save URLs
	_ = storage.Save(ctx, "abc1", "https://example1.com", "user1")
	_ = storage.Save(ctx, "abc2", "https://example2.com", "user1")
	_ = storage.Save(ctx, "abc3", "https://example3.com", "user2")

	// Delete URLs for user1
	err := storage.BatchDelete(ctx, []string{"abc1", "abc2"}, "user1")
	assert.NoError(t, err)

	// Check that URLs are marked as deleted
	_, err = storage.Get(ctx, "abc1")
	assert.ErrorIs(t, err, ErrURLDeleted)

	_, err = storage.Get(ctx, "abc2")
	assert.ErrorIs(t, err, ErrURLDeleted)

	// Check that user2's URL is not affected
	url3, err := storage.Get(ctx, "abc3")
	assert.NoError(t, err)
	assert.Equal(t, "https://example3.com", url3)

	// Test deleting non-existent URLs (should not error)
	err = storage.BatchDelete(ctx, []string{"nonexistent"}, "user1")
	assert.NoError(t, err)
}

func TestMemoryStorage_ConflictDetection(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMemoryStorage(logger)

	ctx := context.Background()

	// Save URL first
	err := storage.Save(ctx, "abc123", "https://example.com", "user1")
	assert.NoError(t, err)

	// Try to save same original URL with different short URL
	err = storage.Save(ctx, "xyz456", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrOriginalURLConflict)

	// Try to save different original URL with same short URL (should be allowed in current implementation)
	err = storage.Save(ctx, "abc123", "https://different.com", "user1")
	assert.NoError(t, err) // This overwrites the previous entry
}

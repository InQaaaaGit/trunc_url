package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func createTempFile(t *testing.T) string {
	tempDir := t.TempDir()
	return filepath.Join(tempDir, "test_urls.json")
}

func TestFileStorage_Save(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Test successful save
	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	assert.NoError(t, err)

	// Test retrieving saved URL
	originalURL, err := storage.Get(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", originalURL)
}

func TestFileStorage_Get(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting non-existent URL
	_, err = storage.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrURLNotFound)

	// Save and then get URL
	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	originalURL, err := storage.Get(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", originalURL)
}

func TestFileStorage_Persistence(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	// Create first storage instance and save data
	storage1, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = storage1.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	// Create second storage instance from same file
	storage2, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	// Verify data is persisted
	originalURL, err := storage2.Get(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", originalURL)
}

func TestFileStorage_GetShortURLByOriginal(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Save URL first
	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	// Test getting short URL by original
	shortURL, err := storage.GetShortURLByOriginal(ctx, "https://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", shortURL)

	// Test getting non-existent original URL
	_, err = storage.GetShortURLByOriginal(ctx, "https://nonexistent.com")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestFileStorage_SaveBatch(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	batch := []BatchEntry{
		{ShortURL: "abc1", OriginalURL: "https://example1.com"},
		{ShortURL: "abc2", OriginalURL: "https://example2.com"},
	}

	// Test successful batch save
	err = storage.SaveBatch(ctx, batch)
	assert.NoError(t, err)

	// Verify saved URLs
	url1, err := storage.Get(ctx, "abc1")
	assert.NoError(t, err)
	assert.Equal(t, "https://example1.com", url1)

	url2, err := storage.Get(ctx, "abc2")
	assert.NoError(t, err)
	assert.Equal(t, "https://example2.com", url2)
}

func TestFileStorage_GetUserURLs(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Save URLs for different users
	_ = storage.Save(ctx, "abc1", "https://example1.com", "user1")
	_ = storage.Save(ctx, "abc2", "https://example2.com", "user1")
	_ = storage.Save(ctx, "abc3", "https://example3.com", "user2")

	// Get URLs for user1
	urls, err := storage.GetUserURLs(ctx, "user1")
	assert.NoError(t, err)
	assert.Len(t, urls, 2)

	// Get URLs for non-existent user
	urls, err = storage.GetUserURLs(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Len(t, urls, 0)
}

func TestFileStorage_BatchDelete(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Save URLs
	_ = storage.Save(ctx, "abc1", "https://example1.com", "user1")
	_ = storage.Save(ctx, "abc2", "https://example2.com", "user1")

	// Delete URLs
	err = storage.BatchDelete(ctx, []string{"abc1", "abc2"}, "user1")
	assert.NoError(t, err)

	// Check that URLs are marked as deleted
	_, err = storage.Get(ctx, "abc1")
	assert.ErrorIs(t, err, ErrURLDeleted)
}

func TestFileStorage_ConflictDetection(t *testing.T) {
	logger := zap.NewNop()
	tempFile := createTempFile(t)

	storage, err := NewFileStorage(tempFile, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Save URL first
	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	// Try to save same original URL with different short URL
	err = storage.Save(ctx, "xyz456", "https://example.com", "user1")
	assert.ErrorIs(t, err, ErrOriginalURLConflict)
}

func TestFileStorage_NewFileStorageErrors(t *testing.T) {
	logger := zap.NewNop()

	// Test with invalid file path (directory doesn't exist)
	invalidPath := "/nonexistent/directory/file.json"
	_, err := NewFileStorage(invalidPath, logger)
	assert.Error(t, err)

	// Test with directory instead of file
	tempDir := t.TempDir()
	_, err = NewFileStorage(tempDir, logger)
	assert.Error(t, err)
}

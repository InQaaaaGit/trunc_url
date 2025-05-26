package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// testUserID используется в тестах как ID пользователя
const testUserID = "test_user_id"

// withTestUserID создает контекст с тестовым ID пользователя
func withTestUserID(ctx context.Context) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, testUserID)
}

func setupTestService(t *testing.T) (*URLServiceImpl, func()) {
	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
	}
	logger, _ := zap.NewDevelopment()
	service, err := NewURLService(cfg, logger)
	if err != nil {
		t.Fatalf("Error creating service: %v", err)
	}
	if service == nil {
		t.Fatal("Service is nil")
	}

	// Приводим к конкретному типу для тестов
	impl, ok := service.(*URLServiceImpl)
	if !ok {
		t.Fatal("service should be of type *URLServiceImpl")
	}

	return impl, func() {
		logger.Sync()
	}
}

func TestConcurrentAccess(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()
	iterations := 100
	errChan := make(chan error, iterations*2)
	logChan := make(chan string, iterations*2)
	done := make(chan struct{})
	ctx := withTestUserID(context.Background())

	// Горутина для логирования
	go func() {
		for msg := range logChan {
			t.Log(msg)
		}
		close(done)
	}()

	// Создаем несколько URL для чтения
	shortIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		shortID, err := service.CreateShortURL(ctx, fmt.Sprintf("https://example%d.com", i))
		if err != nil {
			t.Fatalf("Error creating initial short URL: %v", err)
		}
		shortIDs[i] = shortID
	}

	// Используем отдельную WaitGroup для горутин
	var opsWg sync.WaitGroup

	// Тест конкурентной записи
	opsWg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer opsWg.Done()
			_, err := service.CreateShortURL(ctx, fmt.Sprintf("https://concurrent%d.com", i))
			if err != nil {
				select {
				case errChan <- err:
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: %v", err)
				}
			}
		}(i)
	}

	// Тест конкурентного чтения
	opsWg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer opsWg.Done()
			shortID := shortIDs[i%len(shortIDs)]
			_, err := service.GetOriginalURL(ctx, shortID)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("URL not found for shortID: %s", shortID):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: URL not found for shortID: %s", shortID)
				}
			}
		}(i)
	}

	// Ждем завершения всех операций
	opsWg.Wait()

	// Закрываем каналы после завершения всех операций
	close(errChan)
	close(logChan)

	// Проверяем наличие ошибок в основной горутине
	for err := range errChan {
		t.Errorf("Error during concurrent access: %v", err)
	}

	// Ждем завершения логирования
	<-done
}

func TestConcurrentReadWrite(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()
	iterations := 100
	errChan := make(chan error, iterations*2)
	logChan := make(chan string, iterations*2)
	done := make(chan struct{})
	ctx := withTestUserID(context.Background())

	// Горутина для логирования
	go func() {
		for msg := range logChan {
			t.Log(msg)
		}
		close(done)
	}()

	// Создаем несколько URL для чтения
	shortIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		shortID, err := service.CreateShortURL(ctx, fmt.Sprintf("https://example%d.com", i))
		if err != nil {
			t.Fatalf("Error creating initial short URL: %v", err)
		}
		shortIDs[i] = shortID
	}

	// Используем отдельную WaitGroup для горутин
	var opsWg sync.WaitGroup

	// Тест конкурентного чтения и записи
	opsWg.Add(iterations * 2)
	for i := 0; i < iterations; i++ {
		// Чтение
		go func(i int) {
			defer opsWg.Done()
			shortID := shortIDs[i%len(shortIDs)]
			originalURL, err := service.GetOriginalURL(ctx, shortID)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("URL not found for shortID: %s", shortID):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: URL not found for shortID: %s", shortID)
				}
				return
			}
			expectedURL := fmt.Sprintf("https://example%d.com", i%len(shortIDs))
			if originalURL != expectedURL {
				select {
				case errChan <- fmt.Errorf("unexpected URL for shortID %s: got %s, want %s", shortID, originalURL, expectedURL):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: unexpected URL for shortID %s", shortID)
				}
			}
		}(i)

		// Запись
		go func(i int) {
			defer opsWg.Done()
			shortID, err := service.CreateShortURL(ctx, fmt.Sprintf("https://concurrent%d.com", i))
			if err != nil {
				select {
				case errChan <- fmt.Errorf("error creating short URL: %v", err):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: %v", err)
				}
				return
			}
			// Проверяем, что созданный URL действительно существует
			if _, err := service.GetOriginalURL(ctx, shortID); err != nil {
				select {
				case errChan <- fmt.Errorf("newly created URL %s not found", shortID):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: newly created URL %s not found", shortID)
				}
			}
		}(i)
	}

	// Ждем завершения всех операций
	opsWg.Wait()

	// Закрываем каналы после завершения всех операций
	close(errChan)
	close(logChan)

	// Проверяем наличие ошибок в основной горутине
	for err := range errChan {
		t.Errorf("Error during concurrent read/write: %v", err)
	}

	// Ждем завершения логирования
	<-done
}

func TestCreateShortURL(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := withTestUserID(context.Background())

	tests := []struct {
		name        string
		originalURL string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid URL",
			originalURL: "https://example.com",
			wantErr:     false,
		},
		{
			name:        "Empty URL",
			originalURL: "",
			wantErr:     true,
			errMsg:      "empty URL",
		},
		{
			name:        "Invalid URL",
			originalURL: "not-a-url",
			wantErr:     true,
			errMsg:      "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortURL, err := service.CreateShortURL(ctx, tt.originalURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if shortURL == "" {
				t.Error("expected non-empty short URL")
			}
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := withTestUserID(context.Background())

	// Сначала создаем URL
	originalURL := "https://example.com"
	shortURL, err := service.CreateShortURL(ctx, originalURL)
	if err != nil {
		t.Fatalf("Error creating URL: %v", err)
	}
	if shortURL == "" {
		t.Fatal("Expected non-empty short URL")
	}

	tests := []struct {
		name     string
		shortURL string
		wantURL  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Existing URL",
			shortURL: shortURL,
			wantURL:  originalURL,
			wantErr:  false,
		},
		{
			name:     "Non-existing URL",
			shortURL: "nonexistent",
			wantErr:  true,
			errMsg:   "URL not found",
		},
		{
			name:     "Empty short URL",
			shortURL: "",
			wantErr:  true,
			errMsg:   "empty short URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := service.GetOriginalURL(ctx, tt.shortURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Errorf("expected URL %q, got %q", tt.wantURL, gotURL)
			}
		})
	}
}

func TestCreateShortURLsBatch(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := withTestUserID(context.Background())

	tests := []struct {
		name      string
		batch     []models.BatchRequestEntry
		wantCount int
		wantErr   bool
		errMsg    string
	}{
		{
			name: "Valid batch",
			batch: []models.BatchRequestEntry{
				{CorrelationID: "1", OriginalURL: "https://example1.com"},
				{CorrelationID: "2", OriginalURL: "https://example2.com"},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "Empty batch",
			batch:     []models.BatchRequestEntry{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "Batch with empty URL",
			batch: []models.BatchRequestEntry{
				{CorrelationID: "1", OriginalURL: ""},
				{CorrelationID: "2", OriginalURL: "https://example2.com"},
			},
			wantCount: 1,
			wantErr:   true,
			errMsg:    "all URLs in the batch were invalid or empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBatch, err := service.CreateShortURLsBatch(ctx, tt.batch)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(gotBatch) != tt.wantCount {
				t.Errorf("expected %d URLs, got %d", tt.wantCount, len(gotBatch))
			}
		})
	}
}

func TestURLConflict(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := withTestUserID(context.Background())

	// Создаем первый URL
	originalURL := "https://example.com"
	shortURL1, err := service.CreateShortURL(ctx, originalURL)
	if err != nil {
		t.Fatalf("Error creating first URL: %v", err)
	}
	if shortURL1 == "" {
		t.Fatal("Expected non-empty short URL")
	}

	// Пытаемся создать тот же URL снова
	shortURL2, err := service.CreateShortURL(ctx, originalURL)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, storage.ErrOriginalURLConflict) {
		t.Errorf("expected ErrOriginalURLConflict, got %v", err)
	}
	if shortURL2 != shortURL1 {
		t.Errorf("expected short URL %q, got %q", shortURL1, shortURL2)
	}
}

func TestGetStorage(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	store := service.GetStorage()
	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestMemoryStorage(t *testing.T) {
	ctx := withTestUserID(context.Background())
	logger, _ := zap.NewDevelopment()
	store := storage.NewMemoryStorage(logger)

	// Test Save
	err := store.Save(ctx, "test1", "https://example1.com")
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Test Get
	got, err := store.Get(ctx, "test1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got != "https://example1.com" {
		t.Errorf("Get() = %v, want %v", got, "https://example1.com")
	}

	// Test GetShortURLByOriginal
	shortURL, err := store.GetShortURLByOriginal(ctx, "https://example1.com")
	if err != nil {
		t.Errorf("GetShortURLByOriginal() error = %v", err)
	}
	if shortURL != "test1" {
		t.Errorf("GetShortURLByOriginal() = %v, want %v", shortURL, "test1")
	}

	// Test SaveBatch
	batch := []storage.BatchEntry{
		{ShortURL: "test2", OriginalURL: "https://example2.com", UserID: testUserID},
		{ShortURL: "test3", OriginalURL: "https://example3.com", UserID: testUserID},
	}
	err = store.SaveBatch(ctx, batch)
	if err != nil {
		t.Errorf("SaveBatch() error = %v", err)
	}

	// Test Get after batch save
	got, err = store.Get(ctx, "test2")
	if err != nil {
		t.Errorf("Get() after batch save error = %v", err)
	}
	if got != "https://example2.com" {
		t.Errorf("Get() after batch save = %v, want %v", got, "https://example2.com")
	}

	// Test GetUserURLs
	urls, err := store.GetUserURLs(ctx, testUserID)
	if err != nil {
		t.Errorf("GetUserURLs() error = %v", err)
	}
	if len(urls) != 3 {
		t.Errorf("GetUserURLs() returned %d URLs, want 3", len(urls))
	}
}

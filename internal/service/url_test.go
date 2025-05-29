package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestService(t *testing.T) (*URLServiceImpl, func()) {
	cfg := &config.Config{
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		SecretKey:       "test-secret-key",
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
	ctx := context.Background()
	// Добавляем userID в контекст для тестов, так как CreateShortURL теперь его ожидает
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "test-user-for-create")

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
	ctxWithUser := context.WithValue(context.Background(), middleware.ContextKeyUserID, "test-user-for-concurrent-ops")

	// Тест конкурентной записи
	opsWg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer opsWg.Done()
			_, err := service.CreateShortURL(ctxWithUser, fmt.Sprintf("https://concurrent%d.com", i))
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
	ctx := context.Background()
	// Добавляем userID в контекст для тестов, так как CreateShortURL теперь его ожидает
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "test-user-for-create")

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
	ctxWithUser := context.WithValue(context.Background(), middleware.ContextKeyUserID, "test-user-for-concurrent-rw")

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
			shortID, err := service.CreateShortURL(ctxWithUser, fmt.Sprintf("https://concurrent%d.com", i))
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

	ctx := context.Background()
	// Добавляем userID в контекст для тестов, так как CreateShortURL теперь его ожидает
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "test-user-for-create")

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

	// Создаем URL для теста
	createCtx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "user-for-get-original")
	originalURLToTest := "https://get-original.example.com"
	shortURLToTest, err := service.CreateShortURL(createCtx, originalURLToTest)
	assert.NoError(t, err, "Создание URL для теста GetOriginalURL не должно вызывать ошибку")
	assert.NotEmpty(t, shortURLToTest, "Короткий URL не должен быть пустым")

	// Контекст для самого GetOriginalURL не обязательно должен содержать userID,
	// если логика GetOriginalURL не зависит от пользователя (что сейчас так и есть).
	getCtx := context.Background()

	tests := []struct {
		name     string
		shortURL string
		wantURL  string
		wantErr  bool
		errType  error // Ожидаемый тип ошибки, если wantErr == true
	}{
		{
			name:     "Existing URL",
			shortURL: shortURLToTest,
			wantURL:  originalURLToTest,
			wantErr:  false,
		},
		{
			name:     "Non-existent URL",
			shortURL: "nonexistent",
			wantErr:  true,
			errType:  storage.ErrURLNotFound,
		},
		{
			name:     "Empty short URL",
			shortURL: "",
			wantErr:  true,
			// errType:  fmt.Errorf("empty short URL"), // Точная ошибка зависит от реализации, можем не проверять тип
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := service.GetOriginalURL(getCtx, tt.shortURL)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "Ожидался тип ошибки %T, получена %T (%v)", tt.errType, err, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantURL, gotURL)
			}
		})
	}
}

func TestCreateShortURLsBatch(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

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

	// Создадим URL с userID для теста конфликта
	firstCtx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "user1-conflict-test")
	originalURL := "http://conflict.example.com"
	_, err := service.CreateShortURL(firstCtx, originalURL)
	assert.NoError(t, err, "Первое создание URL не должно вызывать ошибку")

	// Попытка создать тот же URL с тем же userID (если бы GetShortURLByOriginal учитывал userID)
	// или просто тот же originalURL, что вызовет конфликт в GetShortURLByOriginal, если он не учитывает userID
	// Текущая реализация GetShortURLByOriginal не зависит от userID, она ищет глобально.
	// А Save теперь сохраняет с userID. ErrOriginalURLConflict вернется, если GetShortURLByOriginal найдет URL.
	secondCtx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "user2-conflict-test") // Другой или тот же userID
	_, err = service.CreateShortURL(secondCtx, originalURL)
	assert.Error(t, err, "Второе создание того же URL должно вызывать ошибку")
	assert.True(t, errors.Is(err, storage.ErrOriginalURLConflict), "Ожидалась ошибка конфликта URL")
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
	logger, _ := zap.NewDevelopment()
	store := storage.NewMemoryStorage(logger)
	ctx := context.Background()
	testUserID := "test-user-123"

	// Тестирование Save
	err := store.Save(ctx, "short1", "http://example.com/1", testUserID)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Тестирование Get
	originalURL, err := store.Get(ctx, "short1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got := originalURL; got != "http://example.com/1" {
		t.Errorf("Get() = %v, want %v", got, "http://example.com/1")
	}

	// Тестирование GetUserURLs
	userURLs, err := store.GetUserURLs(ctx, testUserID)
	if err != nil {
		t.Fatalf("GetUserURLs failed: %v", err)
	}
	if len(userURLs) != 1 {
		t.Fatalf("Expected 1 URL for user, got %d", len(userURLs))
	}
	if userURLs[0].ShortURL != "short1" || userURLs[0].OriginalURL != "http://example.com/1" {
		t.Errorf("Unexpected user URL data: got %+v", userURLs[0])
	}

	// Тестирование GetUserURLs для несуществующего пользователя
	noUserURLs, err := store.GetUserURLs(ctx, "non-existent-user")
	if err != nil {
		t.Fatalf("GetUserURLs for non-existent user failed: %v", err)
	}
	if len(noUserURLs) != 0 {
		t.Fatalf("Expected 0 URLs for non-existent user, got %d", len(noUserURLs))
	}
}

func TestFileStorage(t *testing.T) {
	filePath := "test_file_storage.json"
	defer os.Remove(filePath) // Очистка после теста

	logger, _ := zap.NewDevelopment()
	store, err := storage.NewFileStorage(filePath, logger)
	if err != nil {
		t.Fatalf("NewFileStorage failed: %v", err)
	}
	ctx := context.Background()
	testUserID := "file-user-456"

	// Тестирование Save
	err = store.Save(ctx, "fileshort1", "http://file.example.com/1", testUserID)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Тестирование Get
	originalURL, err := store.Get(ctx, "fileshort1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if originalURL != "http://file.example.com/1" {
		t.Errorf("Expected URL %s, got %s", "http://file.example.com/1", originalURL)
	}

	// Тестирование GetUserURLs
	userURLs, err := store.GetUserURLs(ctx, testUserID)
	if err != nil {
		t.Fatalf("GetUserURLs failed: %v", err)
	}
	if len(userURLs) != 1 {
		t.Fatalf("Expected 1 URL for user, got %d, urls: %+v", len(userURLs), userURLs)
	}
	if userURLs[0].ShortURL != "fileshort1" || userURLs[0].OriginalURL != "http://file.example.com/1" {
		t.Errorf("Unexpected user URL data: got %+v", userURLs[0])
	}
}

func TestBatchDeleteURLsFanIn(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-fanin"
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)

	// Создаем много URL для тестирования fan-in паттерна
	urlCount := 15 // Больше чем бatchSize (5), чтобы активировать параллельную обработку
	shortURLs := make([]string, urlCount)

	// Создаем URL
	for i := 0; i < urlCount; i++ {
		originalURL := fmt.Sprintf("https://fanin-test-%d.com", i)
		shortURL, err := service.CreateShortURL(ctx, originalURL)
		if err != nil {
			t.Fatalf("Error creating short URL %d: %v", i, err)
		}
		shortURLs[i] = shortURL
	}

	// Проверяем что все URL созданы и доступны
	for i, shortURL := range shortURLs {
		_, err := service.GetOriginalURL(ctx, shortURL)
		if err != nil {
			t.Errorf("URL %d should exist before deletion: %v", i, err)
		}
	}

	// Удаляем URL используя fan-in паттерн
	err := service.BatchDeleteURLs(ctx, shortURLs, userID)
	if err != nil {
		t.Fatalf("BatchDeleteURLs failed: %v", err)
	}

	// Проверяем что все URL помечены как удаленные
	for i, shortURL := range shortURLs {
		_, err := service.GetOriginalURL(ctx, shortURL)
		if err == nil {
			t.Errorf("URL %d should be deleted: %s", i, shortURL)
		}
		if !errors.Is(err, storage.ErrURLDeleted) {
			t.Errorf("URL %d should return ErrURLDeleted, got: %v", i, err)
		}
	}

	t.Logf("Successfully tested fan-in pattern with %d URLs", urlCount)
}

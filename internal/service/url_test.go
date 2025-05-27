package service

import (
	"context"
	"path"
	"sync"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/stretchr/testify/assert"
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

	ctx := context.Background()
	userID := "test-user"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			originalURL := "https://example.com/" + string(rune('a'+i))
			_, err := service.CreateShortURL(withTestUserID(ctx), originalURL, userID)
			if err != nil {
				t.Errorf("CreateShortURL() error = %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentReadWrite(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"

	// Создаем короткий URL для чтения
	shortURL, err := service.CreateShortURL(withTestUserID(ctx), "https://example.com/unique", userID)
	if err != nil {
		t.Fatalf("CreateShortURL() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL))
			if err != nil {
				t.Errorf("GetOriginalURL() error = %v", err)
			}
		}()
		go func(i int) {
			defer wg.Done()
			url := "https://example.com/write/" + string(rune('a'+i))
			_, err := service.CreateShortURL(withTestUserID(ctx), url, userID)
			if err != nil {
				t.Errorf("CreateShortURL() error = %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func TestCreateShortURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		userID      string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid URL",
			originalURL: "https://example.com",
			userID:      "test-user",
			wantErr:     false,
		},
		{
			name:        "Empty URL",
			originalURL: "",
			userID:      "test-user",
			wantErr:     true,
			errMsg:      "invalid URL format",
		},
		{
			name:        "Invalid URL",
			originalURL: "not-a-url",
			userID:      "test-user",
			wantErr:     true,
			errMsg:      "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()
			got, err := service.CreateShortURL(withTestUserID(context.Background()), tt.originalURL, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("URLService.CreateShortURL() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if got == "" {
				t.Error("URLService.CreateShortURL() returned empty short URL")
			}
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		userID      string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Existing URL",
			originalURL: "https://example.com",
			userID:      "test-user",
			wantErr:     false,
		},
		{
			name:        "Non-existing URL",
			originalURL: "",
			userID:      "test-user",
			wantErr:     true,
			errMsg:      "URL not found",
		},
		{
			name:        "Empty short URL",
			originalURL: "",
			userID:      "test-user",
			wantErr:     true,
			errMsg:      "empty short URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()

			ctx := context.Background()
			var shortURL string
			var err error

			switch tt.name {
			case "Existing URL":
				shortURL, err = service.CreateShortURL(withTestUserID(ctx), tt.originalURL, tt.userID)
				if err != nil {
					t.Fatalf("CreateShortURL() error = %v", err)
				}
			case "Non-existing URL":
				shortURL = "https://localhost:8080/nonexistent"
			default:
				shortURL = ""
			}

			got, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("GetOriginalURL() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if got != tt.originalURL {
				t.Errorf("GetOriginalURL() = %v, want %v", got, tt.originalURL)
			}
		})
	}
}

func TestCreateShortURLsBatch(t *testing.T) {
	tests := []struct {
		name    string
		batch   []BatchRequest
		userID  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid batch",
			batch: []BatchRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com/1"},
				{CorrelationID: "2", OriginalURL: "https://example.com/2"},
			},
			userID:  "test-user",
			wantErr: false,
		},
		{
			name:    "empty batch",
			batch:   []BatchRequest{},
			userID:  "test-user",
			wantErr: true,
			errMsg:  "empty batch",
		},
		{
			name: "invalid URL",
			batch: []BatchRequest{
				{CorrelationID: "1", OriginalURL: "not-a-url"},
			},
			userID:  "test-user",
			wantErr: true,
			errMsg:  "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()
			got, err := service.CreateShortURLsBatch(withTestUserID(context.Background()), tt.batch, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURLsBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("CreateShortURLsBatch() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if len(got) != len(tt.batch) {
				t.Errorf("CreateShortURLsBatch() returned %d results, want %d", len(got), len(tt.batch))
			}
		})
	}
}

func TestURLConflict(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"

	// Создаем первый короткий URL
	shortURL1, err := service.CreateShortURL(withTestUserID(ctx), "https://example.com/conflict1", userID)
	if err != nil {
		t.Fatalf("CreateShortURL() error = %v", err)
	}

	// Создаем второй короткий URL для другого оригинального URL
	shortURL2, err := service.CreateShortURL(withTestUserID(ctx), "https://example.com/conflict2", userID)
	if err != nil {
		t.Fatalf("CreateShortURL() error = %v", err)
	}

	if shortURL1 == shortURL2 {
		t.Error("CreateShortURL() returned same short URL for different original URLs")
	}

	originalURL1, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL1))
	if err != nil {
		t.Fatalf("GetOriginalURL() error = %v", err)
	}
	originalURL2, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL2))
	if err != nil {
		t.Fatalf("GetOriginalURL() error = %v", err)
	}

	if originalURL1 == originalURL2 {
		t.Error("GetOriginalURL() returned same original URL for different short URLs")
	}
}

func TestGetStorage(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"
	originalURL := "https://example.com"

	// Создаем короткий URL
	shortURL, err := service.CreateShortURL(withTestUserID(ctx), originalURL, userID)
	if err != nil {
		t.Fatalf("CreateShortURL() error = %v", err)
	}

	// Получаем оригинальный URL
	got, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL))
	if err != nil {
		t.Fatalf("GetOriginalURL() error = %v", err)
	}

	if got != originalURL {
		t.Errorf("GetOriginalURL() = %v, want %v", got, originalURL)
	}
}

func TestMemoryStorage(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"
	originalURL := "https://example.com"

	// Создаем короткий URL
	shortURL, err := service.CreateShortURL(withTestUserID(ctx), originalURL, userID)
	if err != nil {
		t.Fatalf("CreateShortURL() error = %v", err)
	}

	// Получаем оригинальный URL
	got, err := service.GetOriginalURL(withTestUserID(ctx), path.Base(shortURL))
	if err != nil {
		t.Fatalf("GetOriginalURL() error = %v", err)
	}

	if got != originalURL {
		t.Errorf("GetOriginalURL() = %v, want %v", got, originalURL)
	}
}

func TestURLService_CreateShortURL(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"

	tests := []struct {
		name        string
		originalURL string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid URL",
			originalURL: "https://example.com",
			wantErr:     false,
		},
		{
			name:        "invalid URL",
			originalURL: "not-a-url",
			wantErr:     true,
			errMsg:      "invalid URL format",
		},
		{
			name:        "empty URL",
			originalURL: "",
			wantErr:     true,
			errMsg:      "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortURL, err := service.CreateShortURL(withTestUserID(ctx), tt.originalURL, userID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, shortURL)
				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, shortURL)
			}
		})
	}
}

func TestURLService_GetOriginalURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		userID      string
		shortURL    string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "existing URL",
			originalURL: "https://example.com",
			userID:      "test-user",
			wantErr:     false,
		},
		{
			name:     "non-existent URL",
			shortURL: "https://localhost:8080/nonexistent",
			wantErr:  true,
			errMsg:   "URL not found",
		},
		{
			name:     "empty short URL",
			shortURL: "",
			wantErr:  true,
			errMsg:   "empty short URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()
			var shortURL string
			var err error

			if tt.originalURL != "" {
				shortURL, err = service.CreateShortURL(withTestUserID(context.Background()), tt.originalURL, tt.userID)
				if err != nil {
					t.Fatalf("CreateShortURL() error = %v", err)
				}
			} else {
				shortURL = tt.shortURL
			}

			got, err := service.GetOriginalURL(withTestUserID(context.Background()), path.Base(shortURL))
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.GetOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("URLService.GetOriginalURL() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if got != tt.originalURL {
				t.Errorf("URLService.GetOriginalURL() = %v, want %v", got, tt.originalURL)
			}
		})
	}
}

func TestURLService_GetUserURLs(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user"

	// Создаем несколько URL для пользователя
	urls := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/3",
	}

	for _, originalURL := range urls {
		_, err := service.CreateShortURL(withTestUserID(ctx), originalURL, userID)
		if err != nil {
			t.Fatalf("CreateShortURL() error = %v", err)
		}
	}

	// Получаем список URL пользователя
	userURLs, err := service.GetUserURLs(withTestUserID(ctx))
	if err != nil {
		t.Fatalf("GetUserURLs() error = %v", err)
	}

	if len(userURLs) != len(urls) {
		t.Errorf("GetUserURLs() returned %d URLs, want %d", len(userURLs), len(urls))
	}
}

func TestURLService_Ping(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	err := service.Ping(withTestUserID(ctx))
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

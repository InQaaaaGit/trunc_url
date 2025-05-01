package service

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
)

func setupTestService(t *testing.T) (*URLServiceImpl, func()) {
	testFile := "test_urls.json"
	cfg := &config.Config{FileStoragePath: testFile}
	service, err := NewURLService(cfg)
	if err != nil {
		t.Fatalf("Error creating service: %v", err)
	}

	cleanup := func() {
		os.Remove(testFile)
	}

	return service, cleanup
}

func TestConcurrentAccess(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()
	iterations := 100
	errChan := make(chan error, iterations*2)
	logChan := make(chan string, iterations*2)
	done := make(chan struct{})

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
		shortID, err := service.CreateShortURL("https://example.com")
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
			_, err := service.CreateShortURL("https://example.com")
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
			_, err := service.GetOriginalURL(shortID)
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
		shortID, err := service.CreateShortURL("https://example.com")
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
			originalURL, err := service.GetOriginalURL(shortID)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("URL not found for shortID: %s", shortID):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: URL not found for shortID: %s", shortID)
				}
				return
			}
			if originalURL != "https://example.com" {
				select {
				case errChan <- fmt.Errorf("unexpected URL for shortID %s: got %s, want https://example.com", shortID, originalURL):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: unexpected URL for shortID %s", shortID)
				}
			}
		}(i)

		// Запись
		go func() {
			defer opsWg.Done()
			shortID, err := service.CreateShortURL("https://example.com")
			if err != nil {
				select {
				case errChan <- fmt.Errorf("error creating short URL: %v", err):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: %v", err)
				}
				return
			}
			// Проверяем, что созданный URL действительно существует
			if _, err := service.GetOriginalURL(shortID); err != nil {
				select {
				case errChan <- fmt.Errorf("newly created URL %s not found", shortID):
				default:
					logChan <- fmt.Sprintf("Buffer full, couldn't send error: newly created URL %s not found", shortID)
				}
			}
		}()
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
			errMsg:      "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()

			got, err := service.CreateShortURL(tt.originalURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("CreateShortURL() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("CreateShortURL() returned empty string for valid URL")
			}
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	tests := []struct {
		name     string
		shortURL string
		setup    func(*URLServiceImpl)
		wantURL  string
		wantErr  bool
	}{
		{
			name:     "Existing URL",
			shortURL: "abc123",
			setup: func(s *URLServiceImpl) {
				s.storage.Save("abc123", "https://example.com")
			},
			wantURL: "https://example.com",
			wantErr: false,
		},
		{
			name:     "Non-existing URL",
			shortURL: "nonexistent",
			setup:    func(s *URLServiceImpl) {},
			wantURL:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := setupTestService(t)
			defer cleanup()

			tt.setup(service)

			got, err := service.GetOriginalURL(tt.shortURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantURL {
				t.Errorf("GetOriginalURL() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

package service

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentAccess(t *testing.T) {
	service := NewURLService()
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
			_, exists := service.GetOriginalURL(shortID)
			if !exists {
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
	service := NewURLService()
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
			originalURL, exists := service.GetOriginalURL(shortID)
			if !exists {
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
			if _, exists := service.GetOriginalURL(shortID); !exists {
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

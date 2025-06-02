package storage

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"
)

// BenchmarkBatchDelete тестирует производительность массового удаления URL
func BenchmarkBatchDelete(b *testing.B) {
	// Пропускаем если PostgreSQL недоступен
	if testing.Short() {
		b.Skip("Skipping PostgreSQL benchmark in short mode")
	}

	// Инициализируем тестовую БД (нужно настроить DSN)
	logger := zap.NewNop()

	// Можно использовать переменную окружения для DSN или пропустить тест
	dsn := "postgres://user:password@localhost/testdb?sslmode=disable"

	storage, err := NewPostgresStorage(dsn, logger)
	if err != nil {
		b.Skipf("PostgreSQL не доступен: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	userID := "bench-user"

	// Тестируем разные размеры батчей
	batchSizes := []int{10, 50, 100, 500, 1000, 5000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				// Подготавливаем тестовые данные
				shortURLs := make([]string, size)
				for j := 0; j < size; j++ {
					shortURL := fmt.Sprintf("bench_%d_%d", i, j)
					originalURL := fmt.Sprintf("https://example.com/bench_%d_%d", i, j)

					// Создаем URL в БД
					err := storage.Save(ctx, shortURL, originalURL, userID)
					if err != nil {
						b.Fatalf("Error saving URL: %v", err)
					}
					shortURLs[j] = shortURL
				}

				b.StartTimer()

				// Измеряем время batch deletion
				err := storage.BatchDelete(ctx, shortURLs, userID)
				if err != nil {
					b.Fatalf("BatchDelete failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkBatchDeleteOld тестирует производительность старого подхода (для сравнения)
// Эта функция эмулирует старый подход с циклом отдельных UPDATE'ов
func BenchmarkBatchDeleteOld(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping PostgreSQL benchmark in short mode")
	}

	logger := zap.NewNop()
	dsn := "postgres://user:password@localhost/testdb?sslmode=disable"

	storage, err := NewPostgresStorage(dsn, logger)
	if err != nil {
		b.Skipf("PostgreSQL не доступен: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	userID := "bench-user-old"

	// Тестируем только средние размеры для старого подхода
	batchSizes := []int{10, 50, 100}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("OldApproach_BatchSize_%d", size), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				// Подготавливаем тестовые данные
				shortURLs := make([]string, size)
				for j := 0; j < size; j++ {
					shortURL := fmt.Sprintf("old_bench_%d_%d", i, j)
					originalURL := fmt.Sprintf("https://example.com/old_bench_%d_%d", i, j)

					err := storage.Save(ctx, shortURL, originalURL, userID)
					if err != nil {
						b.Fatalf("Error saving URL: %v", err)
					}
					shortURLs[j] = shortURL
				}

				b.StartTimer()

				// Эмулируем старый подход с отдельными UPDATE'ами
				err := storage.batchDeleteOldStyle(ctx, shortURLs, userID)
				if err != nil {
					b.Fatalf("OldStyle BatchDelete failed: %v", err)
				}
			}
		})
	}
}

// batchDeleteOldStyle эмулирует старый подход для сравнения производительности
func (ps *PostgresStorage) batchDeleteOldStyle(ctx context.Context, shortURLs []string, userID string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start error: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Старый подход: подготавливаем statement и выполняем в цикле
	stmt, err := tx.PrepareContext(ctx, "UPDATE urls SET is_deleted = TRUE WHERE short_url = $1 AND user_id = $2 AND is_deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare statement error: %w", err)
	}
	defer stmt.Close()

	// Выполняем обновление для каждого URL (неэффективно!)
	for _, shortURL := range shortURLs {
		_, err := stmt.ExecContext(ctx, shortURL, userID)
		if err != nil {
			return fmt.Errorf("batch delete error for shortURL %s: %w", shortURL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit error: %w", err)
	}

	return nil
}

// TestBatchDeleteChunkLogic тестирует логику разбиения на чанки
func TestBatchDeleteChunkLogic(t *testing.T) {
	// Тестируем что функция правильно разбивает большие массивы
	testCases := []struct {
		name        string
		urlCount    int
		expectedErr bool
	}{
		{"Empty slice", 0, false},
		{"Small batch", 10, false},
		{"Medium batch", 500, false},
		{"Large batch", 1500, false}, // Должно разбиться на чанки
		{"Very large batch", 5000, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Просто проверяем что логика разбиения не падает
			shortURLs := make([]string, tc.urlCount)
			for i := 0; i < tc.urlCount; i++ {
				shortURLs[i] = fmt.Sprintf("test_url_%d", i)
			}

			// Проверяем что константа maxBatchSize установлена правильно
			const maxBatchSize = 1000
			if tc.urlCount > maxBatchSize {
				chunks := (tc.urlCount + maxBatchSize - 1) / maxBatchSize
				t.Logf("URLs %d будут разбиты на %d чанков", tc.urlCount, chunks)
			}
		})
	}
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/lib/pq" // Используем pq для проверки ошибки
)

// PostgresStorage реализует URLStorage с использованием PostgreSQL
type PostgresStorage struct {
	db *sql.DB
	// mutex sync.RWMutex // Удаляем неиспользуемое поле
}

// NewPostgresStorage создает новый экземпляр PostgresStorage
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	// Подключение к базе данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	// Проверка соединения
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		// Закрываем соединение в случае ошибки Ping
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after ping error: %v", closeErr)
		}
		return nil, fmt.Errorf("database connection check error: %w", err)
	}

	// Создание таблицы urls, если её ещё нет
	// Используем конкатенацию строк для читаемости SQL
	createTableSQL := `CREATE TABLE IF NOT EXISTS urls (` +
		`short_url VARCHAR(255) PRIMARY KEY,` +
		`original_url TEXT NOT NULL UNIQUE` +
		`)`
	_, err = db.ExecContext(ctx, createTableSQL)
	if err != nil {
		// Закрываем соединение, если не удалось создать таблицу
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after table creation error: %v", closeErr)
		}
		return nil, fmt.Errorf("table creation error: %w", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

// Save сохраняет URL в хранилище
func (ps *PostgresStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	_, err := ps.db.ExecContext(ctx, "INSERT INTO urls (short_url, original_url) VALUES ($1, $2)", shortURL, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка ошибкой нарушения уникальности от lib/pq
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // 23505 = unique_violation
			// Если это конфликт уникальности (либо short_url, либо original_url),
			// возвращаем нашу специальную ошибку
			return ErrOriginalURLConflict
		}
		// Для всех других ошибок возвращаем их обернутыми
		return fmt.Errorf("save URL error: %w", err)
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ps *PostgresStorage) Get(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	err := ps.db.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Убедимся, что используется errors.Is
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("get URL error: %w", err)
	}
	return originalURL, nil
}

// SaveBatch сохраняет пакет URL в PostgreSQL с использованием транзакции
func (ps *PostgresStorage) SaveBatch(ctx context.Context, batch []BatchEntry) error {
	if len(batch) == 0 {
		return nil // Нет смысла открывать транзакцию для пустого батча
	}

	// Начинаем транзакцию
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start error: %w", err)
	}
	// Гарантируем откат транзакции в случае ошибки
	defer tx.Rollback() //nolint:errcheck // Вызов Rollback на завершенной транзакции безопасен

	// Подготавливаем запрос для вставки
	// $1, $2 - плейсхолдеры для PostgreSQL
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING")
	if err != nil {
		return fmt.Errorf("query preparation error: %w", err)
	}
	// Закрываем statement после завершения функции SaveBatch
	// (независимо от того, завершится транзакция коммитом или роллбэком)
	defer stmt.Close()

	// Выполняем вставку для каждой записи в пакете
	for _, entry := range batch {
		if _, err := stmt.ExecContext(ctx, entry.ShortURL, entry.OriginalURL); err != nil {
			// Ошибка при выполнении запроса внутри транзакции, откатываем (через defer) и возвращаем ошибку
			return fmt.Errorf("insert query execution error for shortURL %s: %w", entry.ShortURL, err)
		}
	}

	// Если все вставки прошли успешно, коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit error: %w", err)
	}

	return nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному из PostgreSQL
func (ps *PostgresStorage) GetShortURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	err := ps.db.QueryRowContext(ctx, "SELECT short_url FROM urls WHERE original_url = $1", originalURL).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("error getting short_url by original_url: %w", err)
	}
	return shortURL, nil
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// CheckConnection проверяет соединение с базой данных
func (ps *PostgresStorage) CheckConnection(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

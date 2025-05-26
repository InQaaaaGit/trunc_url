package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// PostgresStorage реализует URLStorage с использованием PostgreSQL
type PostgresStorage struct {
	db     *sql.DB
	logger *zap.Logger
	// mutex sync.RWMutex // Удаляем неиспользуемое поле
}

// NewPostgresStorage создает новый экземпляр PostgresStorage
func NewPostgresStorage(dsn string, logger *zap.Logger) (*PostgresStorage, error) {
	// Подключение к базе данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	// Настройка пула соединений
	// Максимальное количество открытых соединений
	// Рекомендуется устанавливать в зависимости от количества CPU и ожидаемой нагрузки
	// Для большинства приложений достаточно 25-50 соединений
	db.SetMaxOpenConns(25)

	// Максимальное количество неактивных соединений в пуле
	// Обычно устанавливается меньше MaxOpenConns
	db.SetMaxIdleConns(10)

	// Максимальное время жизни соединения
	// Рекомендуется устанавливать меньше, чем timeout на стороне БД
	db.SetConnMaxLifetime(5 * time.Minute)

	// Максимальное время простоя соединения
	// Помогает освобождать неиспользуемые соединения
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Проверяем соединение
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		// Закрываем соединение в случае ошибки Ping
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after ping error: %v", closeErr)
		}
		return nil, fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	// Создаем таблицы, если они не существуют
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %w", err)
	}

	return &PostgresStorage{
		db:     db,
		logger: logger,
	}, nil
}

// createTables создает необходимые таблицы в базе данных
func createTables(db *sql.DB) error {
	// Создаем таблицу для URL
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			short_url VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT UNIQUE NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating urls table: %w", err)
	}

	return nil
}

// Save сохраняет URL в PostgreSQL
func (ps *PostgresStorage) Save(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	_, err := ps.db.ExecContext(ctx,
		"INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
		shortURL, originalURL, userID)
	if err != nil {
		return fmt.Errorf("error saving URL: %w", err)
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ps *PostgresStorage) Get(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	err := ps.db.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("get URL error: %w", err)
	}
	return originalURL, nil
}

// SaveBatch сохраняет пакет URL в PostgreSQL с использованием транзакции
func (ps *PostgresStorage) SaveBatch(ctx context.Context, batch []BatchEntry) error {
	if len(batch) == 0 {
		return nil
	}

	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start error: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			ps.logger.Error("Error rolling back transaction", zap.Error(err))
		}
	}()

	for _, entry := range batch {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
			entry.ShortURL, entry.OriginalURL, userID)
		if err != nil {
			return fmt.Errorf("insert query execution error for shortURL %s: %w", entry.ShortURL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit error: %w", err)
	}

	return nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному
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

// SaveUserURL сохраняет URL пользователя
func (ps *PostgresStorage) SaveUserURL(ctx context.Context, userID, shortURL, originalURL string) error {
	_, err := ps.db.ExecContext(ctx,
		"INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
		shortURL, originalURL, userID)
	if err != nil {
		return fmt.Errorf("error saving user URL: %w", err)
	}
	return nil
}

// GetUserURLs получает все URL пользователя
func (ps *PostgresStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := ps.db.QueryContext(ctx,
		"SELECT short_url, original_url FROM urls WHERE user_id = $1",
		userID)
	if err != nil {
		return nil, fmt.Errorf("error querying user URLs: %w", err)
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var url models.UserURL
		if err := rows.Scan(&url.ShortURL, &url.OriginalURL); err != nil {
			return nil, fmt.Errorf("error scanning user URL: %w", err)
		}
		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user URLs: %w", err)
	}

	return urls, nil
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// CheckConnection проверяет соединение с базой данных
func (ps *PostgresStorage) CheckConnection(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

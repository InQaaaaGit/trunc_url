package storage

import (
	"context"
	"fmt"

	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresStorage реализует хранение URL в PostgreSQL
type PostgresStorage struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresStorage создает новый экземпляр PostgresStorage
func NewPostgresStorage(ctx context.Context, dsn string, logger *zap.Logger) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	storage := &PostgresStorage{
		pool:   pool,
		logger: logger,
	}

	// Проверяем соединение
	if err := storage.CheckConnection(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to check connection: %w", err)
	}

	// Создаем таблицы, если они не существуют
	if err := storage.createTables(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// SaveURL сохраняет пару короткий URL - оригинальный URL
func (s *PostgresStorage) SaveURL(ctx context.Context, shortURL, originalURL string) error {
	// Получаем userID из контекста
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return fmt.Errorf("user_id not found in context")
	}

	// Проверяем, существует ли уже такой URL
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM urls WHERE original_url = $1
		)
	`, originalURL).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check URL existence: %w", err)
	}

	if exists {
		return ErrURLAlreadyExists
	}

	// Сохраняем URL
	_, err = s.pool.Exec(ctx, `
		INSERT INTO urls (short_url, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_url) DO NOTHING
	`, shortURL, originalURL, userID)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}

	return nil
}

// GetOriginalURL возвращает оригинальный URL по короткому URL
func (s *PostgresStorage) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	err := s.pool.QueryRow(ctx, `
		SELECT original_url FROM urls WHERE short_url = $1
	`, shortURL).Scan(&originalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("failed to get original URL: %w", err)
	}

	return originalURL, nil
}

// GetShortURL возвращает короткий URL по оригинальному URL
func (s *PostgresStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	err := s.pool.QueryRow(ctx, `
		SELECT short_url FROM urls WHERE original_url = $1
	`, originalURL).Scan(&shortURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("failed to get short URL: %w", err)
	}

	return shortURL, nil
}

// GetUserURLs возвращает список URL пользователя
func (s *PostgresStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT short_url, original_url
		FROM urls
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user URLs: %w", err)
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var url models.UserURL
		if err := rows.Scan(&url.ShortURL, &url.OriginalURL); err != nil {
			return nil, fmt.Errorf("failed to scan URL row: %w", err)
		}
		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating URL rows: %w", err)
	}

	return urls, nil
}

// CheckConnection проверяет соединение с хранилищем
func (s *PostgresStorage) CheckConnection(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close закрывает соединение с хранилищем
func (s *PostgresStorage) Close() error {
	s.pool.Close()
	return nil
}

// createTables создает необходимые таблицы в базе данных
func (s *PostgresStorage) createTables(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			short_url VARCHAR(255) PRIMARY KEY,
			original_url TEXT NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(original_url)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create urls table: %w", err)
	}

	return nil
}

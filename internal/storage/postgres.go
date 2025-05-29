package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/lib/pq" // Используем pq для проверки ошибки
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

	// Создание таблицы urls, если её ещё нет
	// Используем конкатенацию строк для читаемости SQL
	createTableSQL := `CREATE TABLE IF NOT EXISTS urls (` +
		`short_url VARCHAR(255) PRIMARY KEY,` +
		`original_url TEXT NOT NULL,` +
		`user_id VARCHAR(255),` +
		`is_deleted BOOLEAN DEFAULT FALSE,` +
		`CONSTRAINT unique_original_url_per_user UNIQUE (original_url, user_id)` +
		`)`
	_, err = db.ExecContext(ctx, createTableSQL)
	if err != nil {
		// Закрываем соединение, если не удалось создать таблицу
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after table creation error: %v", closeErr)
		}
		return nil, fmt.Errorf("table creation error: %w", err)
	}

	// Добавляем поле is_deleted, если его нет (для существующих таблиц)
	alterTableSQL := `ALTER TABLE urls ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE`
	_, err = db.ExecContext(ctx, alterTableSQL)
	if err != nil {
		logger.Warn("Failed to add is_deleted column (may already exist)", zap.Error(err))
	}

	return &PostgresStorage{
		db:     db,
		logger: logger,
	}, nil
}

// Save сохраняет URL в хранилище, связывая его с userID
func (ps *PostgresStorage) Save(ctx context.Context, shortURL, originalURL, userID string) error {
	_, err := ps.db.ExecContext(ctx, "INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3)", shortURL, originalURL, userID)
	if err != nil {
		// Проверяем, является ли ошибка ошибкой нарушения уникальности от lib/pq
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // 23505 = unique_violation
			// Если это конфликт уникальности (либо short_url, либо original_url + user_id),
			// возвращаем нашу специальную ошибку
			// Здесь может потребоваться более точная проверка, какой именно constraint вызвал конфликт,
			// если original_url должен быть глобально уникальным, а не только для пользователя.
			// В текущей постановке задачи - original_url может повторяться у разных пользователей.
			return ErrOriginalURLConflict // Или другая специфичная ошибка
		}
		// Для всех других ошибок возвращаем их обернутыми
		return fmt.Errorf("save URL error: %w", err)
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ps *PostgresStorage) Get(ctx context.Context, shortURL string) (string, error) {
	var originalURL string
	var isDeleted bool
	err := ps.db.QueryRowContext(ctx, "SELECT original_url, is_deleted FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Убедимся, что используется errors.Is
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("get URL error: %w", err)
	}

	if isDeleted {
		return "", ErrURLDeleted
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

	// Выполняем вставку для каждой записи в пакете
	for _, entry := range batch {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING",
			entry.ShortURL, entry.OriginalURL)
		if err != nil {
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

// GetUserURLs получает все URL, сохраненные пользователем, из PostgreSQL
func (ps *PostgresStorage) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := ps.db.QueryContext(ctx, "SELECT short_url, original_url FROM urls WHERE user_id = $1 AND is_deleted = FALSE", userID)
	if err != nil {
		return nil, fmt.Errorf("query user URLs error: %w", err)
	}
	defer rows.Close()

	var userURLs []models.UserURL
	for rows.Next() {
		var u models.UserURL
		if err := rows.Scan(&u.ShortURL, &u.OriginalURL); err != nil {
			return nil, fmt.Errorf("scan user URL error: %w", err)
		}
		userURLs = append(userURLs, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return userURLs, nil
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// CheckConnection проверяет соединение с базой данных
func (ps *PostgresStorage) CheckConnection(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

// BatchDelete помечает URL как удаленные для указанного пользователя
func (ps *PostgresStorage) BatchDelete(ctx context.Context, shortURLs []string, userID string) error {
	if len(shortURLs) == 0 {
		return nil // Нет смысла открывать транзакцию для пустого списка
	}

	// Начинаем транзакцию
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start error: %w", err)
	}
	// Гарантируем откат транзакции в случае ошибки
	defer tx.Rollback() //nolint:errcheck

	// Подготавливаем запрос для batch update
	stmt, err := tx.PrepareContext(ctx, "UPDATE urls SET is_deleted = TRUE WHERE short_url = $1 AND user_id = $2 AND is_deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare statement error: %w", err)
	}
	defer stmt.Close()

	// Выполняем обновление для каждого URL
	for _, shortURL := range shortURLs {
		_, err := stmt.ExecContext(ctx, shortURL, userID)
		if err != nil {
			return fmt.Errorf("batch delete error for shortURL %s: %w", shortURL, err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit error: %w", err)
	}

	return nil
}

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/lib/pq"   // Используем pq для проверки ошибки
	_ "github.com/lib/pq" // Импорт драйвера PostgreSQL
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
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Проверка соединения
	if err := db.Ping(); err != nil {
		// Закрываем соединение в случае ошибки Ping
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after ping error: %v", closeErr)
		}
		return nil, fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	// Создание таблицы urls, если её ещё нет
	// Используем конкатенацию строк для читаемости SQL
	createTableSQL := `CREATE TABLE IF NOT EXISTS urls (` +
		`short_url VARCHAR(255) PRIMARY KEY,` +
		`original_url TEXT NOT NULL UNIQUE` +
		`)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		// Закрываем соединение, если не удалось создать таблицу
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close DB connection after table creation error: %v", closeErr)
		}
		return nil, fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

// Save сохраняет URL в хранилище
func (ps *PostgresStorage) Save(shortURL, originalURL string) error {
	_, err := ps.db.Exec("INSERT INTO urls (short_url, original_url) VALUES ($1, $2)", shortURL, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка ошибкой нарушения уникальности от lib/pq
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // 23505 = unique_violation
			// Если это конфликт уникальности (либо short_url, либо original_url),
			// возвращаем нашу специальную ошибку
			return ErrOriginalURLConflict
		}
		// Для всех других ошибок возвращаем их обернутыми
		return fmt.Errorf("ошибка сохранения URL: %w", err)
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ps *PostgresStorage) Get(shortURL string) (string, error) {
	var originalURL string
	err := ps.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Убедимся, что используется errors.Is
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("ошибка получения URL: %w", err)
	}
	return originalURL, nil
}

// SaveBatch сохраняет пакет URL в PostgreSQL с использованием транзакции
func (ps *PostgresStorage) SaveBatch(batch []BatchEntry) error {
	if len(batch) == 0 {
		return nil // Нет смысла открывать транзакцию для пустого батча
	}

	// Начинаем транзакцию
	tx, err := ps.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	// Гарантируем откат транзакции в случае ошибки
	defer tx.Rollback() //nolint:errcheck // Вызов Rollback на завершенной транзакции безопасен

	// Подготавливаем запрос для вставки
	// $1, $2 - плейсхолдеры для PostgreSQL
	stmt, err := tx.Prepare("INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING")
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %w", err)
	}
	// Закрываем statement после завершения функции SaveBatch
	// (независимо от того, завершится транзакция коммитом или роллбэком)
	defer stmt.Close()

	// Выполняем вставку для каждой записи в пакете
	for _, entry := range batch {
		if _, err := stmt.Exec(entry.ShortURL, entry.OriginalURL); err != nil {
			// Ошибка при выполнении запроса внутри транзакции, откатываем (через defer) и возвращаем ошибку
			return fmt.Errorf("ошибка выполнения запроса вставки для shortURL %s: %w", entry.ShortURL, err)
		}
	}

	// Если все вставки прошли успешно, коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetShortURLByOriginal получает короткий URL по оригинальному из PostgreSQL
func (ps *PostgresStorage) GetShortURLByOriginal(originalURL string) (string, error) {
	var shortURL string
	err := ps.db.QueryRow("SELECT short_url FROM urls WHERE original_url = $1", originalURL).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("ошибка получения short_url по original_url: %w", err)
	}
	return shortURL, nil
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// CheckConnection проверяет соединение с базой данных
func (ps *PostgresStorage) CheckConnection() error {
	return ps.db.Ping()
}

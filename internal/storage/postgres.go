package storage

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/lib/pq" // Импорт драйвера PostgreSQL
)

// PostgresStorage реализует URLStorage с использованием PostgreSQL
type PostgresStorage struct {
	db    *sql.DB
	mutex sync.RWMutex
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
		db.Close()
		return nil, fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	// Создание таблицы urls, если её ещё нет
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			short_url VARCHAR(255) PRIMARY KEY,
			original_url TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return &PostgresStorage{
		db: db,
	}, nil
}

// Save сохраняет URL в хранилище
func (ps *PostgresStorage) Save(shortURL, originalURL string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	_, err := ps.db.Exec("INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING", shortURL, originalURL)
	if err != nil {
		return fmt.Errorf("ошибка сохранения URL: %w", err)
	}
	return nil
}

// Get получает оригинальный URL по короткому
func (ps *PostgresStorage) Get(shortURL string) (string, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var originalURL string
	err := ps.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("ошибка получения URL: %w", err)
	}
	return originalURL, nil
}

// Close закрывает соединение с базой данных
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

// CheckConnection проверяет соединение с базой данных
func (ps *PostgresStorage) CheckConnection() error {
	return ps.db.Ping()
}

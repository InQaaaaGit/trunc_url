package service

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
)

// URLService определяет интерфейс сервиса для работы с URL
type URLService interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortURL string) (string, error)
	GetStorage() storage.URLStorage
	CreateShortURLsBatch(batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
}

// URLServiceImpl реализует URLService
type URLServiceImpl struct {
	storage storage.URLStorage
	config  *config.Config
}

// NewURLService создает новый экземпляр URLService, выбирая хранилище
// в зависимости от конфигурации с приоритетом: PostgreSQL -> Файл -> Память.
func NewURLService(cfg *config.Config) (*URLServiceImpl, error) {
	var store storage.URLStorage
	var err error

	// 1. Пытаемся использовать PostgreSQL, если задан DSN
	if cfg.DatabaseDSN != "" {
		log.Println("Используется PostgreSQL хранилище:", cfg.DatabaseDSN)
		store, err = storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			// Логируем ошибку, но не выходим, т.к. можем перейти к файловому хранилищу
			log.Printf("Ошибка инициализации PostgreSQL хранилища: %v. Переход к файловому хранилищу.", err)
		} else {
			// Успешно создали PostgresStorage, используем его
			return &URLServiceImpl{
				storage: store,
				config:  cfg,
			}, nil
		}
	}

	// 2. Пытаемся использовать Файловое хранилище, если задан путь и Postgres не удался или не был задан
	if cfg.FileStoragePath != "" {
		log.Println("Используется файловое хранилище:", cfg.FileStoragePath)
		store, err = storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			// Логируем ошибку, но не выходим, т.к. можем перейти к хранилищу в памяти
			log.Printf("Ошибка инициализации файлового хранилища: %v. Переход к хранилищу в памяти.", err)
		} else {
			// Успешно создали FileStorage, используем его
			return &URLServiceImpl{
				storage: store,
				config:  cfg,
			}, nil
		}
	}

	// 3. Используем хранилище в памяти как запасной вариант
	log.Println("Используется хранилище в памяти.")
	store = storage.NewMemoryStorage() // NewMemoryStorage не возвращает ошибку

	return &URLServiceImpl{
		storage: store,
		config:  cfg,
	}, nil // Ошибки здесь быть не может, т.к. MemoryStorage всегда создается успешно
}

// CreateShortURL создает короткий URL из оригинального
func (s *URLServiceImpl) CreateShortURL(originalURL string) (string, error) {
	if originalURL == "" {
		return "", errors.New("empty URL")
	}

	// Проверяем валидность URL
	_, err := url.ParseRequestURI(originalURL)
	if err != nil {
		return "", errors.New("invalid URL")
	}

	hash := sha256.Sum256([]byte(originalURL))
	shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]

	err = s.storage.Save(shortURL, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка конфликтом оригинального URL
		if errors.Is(err, storage.ErrOriginalURLConflict) {
			// Если URL уже существует, получаем существующий shortURL
			log.Printf("Конфликт: Original URL '%s' уже существует. Получаем существующий short URL.", originalURL)
			existingShortURL, getErr := s.storage.GetShortURLByOriginal(originalURL)
			if getErr != nil {
				// Эта ситуация не должна произойти, если Save вернул конфликт,
				// но обрабатываем на всякий случай
				log.Printf("Критическая ошибка: не удалось получить short URL для существующего original URL '%s': %v", originalURL, getErr)
				return "", fmt.Errorf("ошибка получения существующего short URL: %w", getErr)
			}
			// Возвращаем существующий shortURL и ошибку конфликта для обработки в хендлере
			return existingShortURL, storage.ErrOriginalURLConflict
		}

		// Для других ошибок сохранения просто логируем и возвращаем
		log.Printf("Ошибка сохранения URL: %v", err)
		return "", err
	}

	// Если ошибки нет, возвращаем новый shortURL
	return shortURL, nil
}

// GetOriginalURL получает оригинальный URL по короткому
func (s *URLServiceImpl) GetOriginalURL(shortURL string) (string, error) {
	originalURL, err := s.storage.Get(shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("URL не найден для shortID: %s", shortURL)
		} else {
			log.Printf("Ошибка получения URL для shortID %s: %v", shortURL, err)
		}
		return "", err // Возвращаем ошибку (включая ErrURLNotFound)
	}
	return originalURL, nil
}

// CreateShortURLsBatch создает короткие URL для пакета и сохраняет их
func (s *URLServiceImpl) CreateShortURLsBatch(reqBatch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	if len(reqBatch) == 0 {
		return []models.BatchResponseEntry{}, nil // Возвращаем пустой слайс, если на входе пусто
	}

	storageBatch := make([]storage.BatchEntry, 0, len(reqBatch))
	respBatch := make([]models.BatchResponseEntry, 0, len(reqBatch))

	for _, reqEntry := range reqBatch {
		originalURL := reqEntry.OriginalURL
		if originalURL == "" {
			// Пропускаем пустые URL или возвращаем ошибку?
			// По ТЗ вроде не сказано, но логично пропустить или вернуть ошибку на весь батч.
			// Пока пропустим и залогируем.
			log.Printf("Пропущен пустой URL в батче для correlation_id: %s", reqEntry.CorrelationID)
			continue
		}
		// TODO: Добавить валидацию URL, как в CreateShortURL?
		// _, err := url.ParseRequestURI(originalURL)
		// if err != nil { ... }

		// Генерируем shortURL (та же логика, что и в CreateShortURL)
		hash := sha256.Sum256([]byte(originalURL))
		shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]
		fullShortURL := s.config.BaseURL + "/" + shortURL // Формируем полный URL для ответа

		// Добавляем в батч для сохранения в хранилище
		storageBatch = append(storageBatch, storage.BatchEntry{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})

		// Добавляем в батч для ответа
		respBatch = append(respBatch, models.BatchResponseEntry{
			CorrelationID: reqEntry.CorrelationID,
			ShortURL:      fullShortURL,
		})
	}

	// Если после фильтрации пустых URL батч для хранилища опустел
	if len(storageBatch) == 0 {
		log.Println("Пакет для сохранения пуст после обработки входных данных.")
		// Возвращаем пустой respBatch, так как валидных URL не было
		return []models.BatchResponseEntry{}, errors.New("все URL в пакете были невалидны или пусты")
	}

	// Сохраняем весь батч в хранилище
	err := s.storage.SaveBatch(storageBatch)
	if err != nil {
		log.Printf("Ошибка сохранения пакета URL: %v", err)
		return nil, fmt.Errorf("ошибка сохранения пакета: %w", err) // Возвращаем ошибку
	}

	// Возвращаем результат
	return respBatch, nil
}

// GetStorage возвращает хранилище URL
func (s *URLServiceImpl) GetStorage() storage.URLStorage {
	return s.storage
}

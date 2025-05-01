package service

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
)

// URLService определяет интерфейс сервиса для работы с URL
type URLService interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortURL string) (string, error)
	GetStorage() storage.URLStorage
}

// URLServiceImpl реализует URLService
type URLServiceImpl struct {
	storage storage.URLStorage
	config  *config.Config
}

// NewURLService создает новый экземпляр URLService
func NewURLService(cfg *config.Config) (*URLServiceImpl, error) {
	var store storage.URLStorage
	var err error

	// Проверяем, указана ли строка подключения к БД
	if cfg.DatabaseDSN != "" {
		// Создаем хранилище PostgreSQL
		store, err = storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, err
		}
	} else {
		// Создаем файловое хранилище, если строка подключения не указана
		store, err = storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}
	}

	return &URLServiceImpl{
		storage: store,
		config:  cfg,
	}, nil
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
		return "", err
	}

	return shortURL, nil
}

// GetOriginalURL получает оригинальный URL по короткому
func (s *URLServiceImpl) GetOriginalURL(shortURL string) (string, error) {
	return s.storage.Get(shortURL)
}

// GetStorage возвращает хранилище URL
func (s *URLServiceImpl) GetStorage() storage.URLStorage {
	return s.storage
}

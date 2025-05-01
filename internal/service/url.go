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
}

// URLServiceImpl реализует URLService
type URLServiceImpl struct {
	storage storage.URLStorage
	config  *config.Config
}

// NewURLService создает новый экземпляр URLService
func NewURLService(cfg *config.Config) (*URLServiceImpl, error) {
	// Создаем файловое хранилище
	fileStorage, err := storage.NewFileStorage(cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}

	return &URLServiceImpl{
		storage: fileStorage,
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

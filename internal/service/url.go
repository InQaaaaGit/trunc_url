package service

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"

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
}

// NewURLService создает новый экземпляр URLService
func NewURLService() *URLServiceImpl {
	return &URLServiceImpl{
		storage: storage.NewMemoryStorage(),
	}
}

// CreateShortURL создает короткий URL из оригинального
func (s *URLServiceImpl) CreateShortURL(originalURL string) (string, error) {
	if originalURL == "" {
		return "", errors.New("empty URL")
	}
	hash := sha256.Sum256([]byte(originalURL))
	shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]

	err := s.storage.Save(shortURL, originalURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

// GetOriginalURL получает оригинальный URL по короткому
func (s *URLServiceImpl) GetOriginalURL(shortURL string) (string, error) {
	return s.storage.Get(shortURL)
}

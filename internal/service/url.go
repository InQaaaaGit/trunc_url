package service

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

type URLService interface {
	CreateShortURL(url string) (string, error)
	GetOriginalURL(shortID string) (string, bool)
}

type URLServiceImpl struct {
	urls map[string]string
}

func NewURLService() URLService {
	return &URLServiceImpl{
		urls: make(map[string]string),
	}
}

func (s *URLServiceImpl) GenerateShortID() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

func (s *URLServiceImpl) CreateShortURL(originalURL string) (string, error) {
	log.Printf("Создание короткой ссылки для: %s", originalURL)
	shortID, err := s.GenerateShortID()
	if err != nil {
		return "", err
	}

	s.urls[shortID] = originalURL
	log.Printf("Создано соответствие: %s -> %s", shortID, originalURL)
	log.Printf("Текущее состояние urls: %v", s.urls)
	return shortID, nil
}

func (s *URLServiceImpl) GetOriginalURL(shortID string) (string, bool) {
	log.Printf("Поиск URL для shortID: %s", shortID)
	log.Printf("Текущее состояние urls: %v", s.urls)
	originalURL, exists := s.urls[shortID]
	log.Printf("Результат поиска - URL: %s, существует: %v", originalURL, exists)
	return originalURL, exists
}

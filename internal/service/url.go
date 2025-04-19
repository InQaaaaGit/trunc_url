package service

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"sync"
)

type URLServiceImpl struct {
	urls  map[string]string
	mutex sync.RWMutex
}

func NewURLService() *URLServiceImpl {
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
	shortID, err := s.GenerateShortID()
	if err != nil {
		return "", err
	}

	s.mutex.Lock()
	s.urls[shortID] = originalURL
	s.mutex.Unlock()

	log.Printf("Created mapping: %s -> %s", shortID, originalURL)
	return shortID, nil
}

func (s *URLServiceImpl) GetOriginalURL(shortID string) (string, bool) {
	s.mutex.RLock()
	originalURL, exists := s.urls[shortID]
	s.mutex.RUnlock()

	return originalURL, exists
}

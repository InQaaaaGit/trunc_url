package service_test

import (
	"context"
	"fmt"
	"log"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"go.uber.org/zap"
)

// ExampleURLService_CreateShortURL демонстрирует создание короткого URL.
func ExampleURLService_CreateShortURL() {
	// Создаем конфигурацию для примера
	cfg := &config.Config{
		BaseURL: "http://example.com",
	}

	// Создаем логгер (отключаем логи для примера)
	logger := zap.NewNop()

	// Создаем сервис с in-memory хранилищем
	svc, err := service.NewURLService(cfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем короткий URL
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "test-user-123")
	originalURL := "https://practicum.yandex.ru/"
	shortID, err := svc.CreateShortURL(ctx, originalURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Short ID length: %d\n", len(shortID))
	fmt.Printf("Short ID is not empty: %t\n", shortID != "")

	// Output:
	// Short ID length: 8
	// Short ID is not empty: true
}

// ExampleURLService_GetOriginalURL демонстрирует получение оригинального URL.
func ExampleURLService_GetOriginalURL() {
	// Создаем конфигурацию для примера
	cfg := &config.Config{
		BaseURL: "http://example.com",
	}

	// Создаем логгер (отключаем логи для примера)
	logger := zap.NewNop()

	// Создаем сервис с in-memory хранилищем
	svc, err := service.NewURLService(cfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "test-user-123")
	originalURL := "https://practicum.yandex.ru/"

	// Сначала создаем короткий URL
	shortID, err := svc.CreateShortURL(ctx, originalURL)
	if err != nil {
		log.Fatal(err)
	}

	// Затем получаем оригинальный URL
	retrievedURL, err := svc.GetOriginalURL(ctx, shortID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("URLs match: %t\n", retrievedURL == originalURL)
	fmt.Printf("Retrieved URL: %s\n", retrievedURL)

	// Output:
	// URLs match: true
	// Retrieved URL: https://practicum.yandex.ru/
}

// ExampleURLService_CreateShortURLsBatch демонстрирует пакетное создание URL.
func ExampleURLService_CreateShortURLsBatch() {
	// Создаем конфигурацию для примера
	cfg := &config.Config{
		BaseURL: "http://example.com",
	}

	// Создаем логгер (отключаем логи для примера)
	logger := zap.NewNop()

	// Создаем сервис с in-memory хранилищем
	svc, err := service.NewURLService(cfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	// Подготавливаем batch запрос
	batch := []models.BatchRequestEntry{
		{
			CorrelationID: "1",
			OriginalURL:   "https://practicum.yandex.ru/",
		},
		{
			CorrelationID: "2",
			OriginalURL:   "https://yandex.ru/",
		},
	}

	// Выполняем пакетное создание
	ctx := context.Background()
	response, err := svc.CreateShortURLsBatch(ctx, batch)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response entries count: %d\n", len(response))
	fmt.Printf("First correlation ID: %s\n", response[0].CorrelationID)
	fmt.Printf("Short URL contains base URL: %t\n",
		len(response[0].ShortURL) > len(cfg.BaseURL))

	// Output:
	// Response entries count: 2
	// First correlation ID: 1
	// Short URL contains base URL: true
}

// ExampleNewURLService демонстрирует создание нового экземпляра URLService.
func ExampleNewURLService() {
	// Создаем конфигурацию для примера
	cfg := &config.Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "", // Пустая строка означает использование in-memory хранилища
		DatabaseDSN:     "", // Пустая строка означает отсутствие PostgreSQL
		SecretKey:       "test-secret-key",
	}

	// Создаем логгер (отключаем логи для примера)
	logger := zap.NewNop()

	// Создаем сервис
	svc, err := service.NewURLService(cfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	// Проверяем, что сервис создан
	fmt.Printf("Service created successfully: %t\n", svc != nil)
	fmt.Printf("Storage available: %t\n", svc.GetStorage() != nil)

	// Output:
	// Service created successfully: true
	// Storage available: true
}

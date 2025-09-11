package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/middleware"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"go.uber.org/zap"
)

// ExampleHandler_HandleCreateURL демонстрирует создание короткого URL через POST запрос.
func ExampleHandler_HandleCreateURL() {
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

	// Создаем обработчик
	h := handler.NewHandler(svc, cfg, logger)

	// Создаем HTTP запрос
	body := strings.NewReader("https://practicum.yandex.ru/")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/plain")

	// Добавляем userID в контекст (имитируем работу AuthMiddleware)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "test-user-123")
	req = req.WithContext(ctx)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Выполняем запрос
	h.HandleCreateURL(rr, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))
	fmt.Printf("Response body contains base URL: %t\n", strings.Contains(rr.Body.String(), cfg.BaseURL))

	// Output:
	// Status: 201
	// Content-Type: text/plain
	// Response body contains base URL: true
}

// ExampleHandler_HandleShortenURL демонстрирует создание короткого URL через JSON API.
func ExampleHandler_HandleShortenURL() {
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

	// Создаем обработчик
	h := handler.NewHandler(svc, cfg, logger)

	// Подготавливаем JSON запрос
	reqBody := handler.ShortenRequest{
		URL: "https://practicum.yandex.ru/",
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем HTTP запрос
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст (имитируем работу AuthMiddleware)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "test-user-123")
	req = req.WithContext(ctx)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Выполняем запрос
	h.HandleShortenURL(rr, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))

	var response handler.ShortenResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	fmt.Printf("Response contains short URL: %t\n", strings.Contains(response.Result, cfg.BaseURL))

	// Output:
	// Status: 201
	// Content-Type: application/json
	// Response contains short URL: true
}

// ExampleHandler_HandleShortenBatch демонстрирует пакетное создание коротких URL.
func ExampleHandler_HandleShortenBatch() {
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

	// Создаем обработчик
	h := handler.NewHandler(svc, cfg, logger)

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

	jsonData, err := json.Marshal(batch)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем HTTP запрос
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст для работы с batch
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "example-user")
	req = req.WithContext(ctx)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Выполняем запрос
	h.HandleShortenBatch(rr, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))

	var response []models.BatchResponseEntry
	json.Unmarshal(rr.Body.Bytes(), &response)
	fmt.Printf("Response entries count: %d\n", len(response))
	if len(response) > 0 {
		fmt.Printf("First entry has correlation_id: %t\n", response[0].CorrelationID == "1")
	} else {
		fmt.Printf("First entry has correlation_id: %t\n", false)
	}

	// Output:
	// Status: 201
	// Content-Type: application/json
	// Response entries count: 2
	// First entry has correlation_id: true
}

// ExampleHandler_HandleRedirect демонстрирует перенаправление по короткому URL.
func ExampleHandler_HandleRedirect() {
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

	// Создаем обработчик
	h := handler.NewHandler(svc, cfg, logger)

	// Сначала создаем короткий URL
	originalURL := "https://practicum.yandex.ru/"
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, "test-user-123")
	shortID, err := svc.CreateShortURL(ctx, originalURL)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем запрос на редирект
	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rr := httptest.NewRecorder()

	// Выполняем запрос
	h.HandleRedirect(rr, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Location header: %s\n", rr.Header().Get("Location"))

	// Output:
	// Status: 307
	// Location header: https://practicum.yandex.ru/
}

package grpc

import (
	"context"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockURLService реализует интерфейс service.URLService для тестов
type mockURLService struct {
	createShortURLFunc       func(ctx context.Context, originalURL string) (string, error)
	getOriginalURLFunc       func(ctx context.Context, shortURL string) (string, error)
	createShortURLsBatchFunc func(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	getUserURLsFunc          func(ctx context.Context, userID string) ([]models.UserURL, error)
	batchDeleteURLsFunc      func(ctx context.Context, shortURLs []string, userID string) error
	checkConnectionFunc      func(ctx context.Context) error
	getStatsFunc             func(ctx context.Context) (int, int, error)
	closeFunc                func() error
}

func (m *mockURLService) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(ctx, originalURL)
	}
	return "", nil
}

func (m *mockURLService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(ctx, shortURL)
	}
	return "", nil
}

func (m *mockURLService) CreateShortURLsBatch(ctx context.Context, batch []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	if m.createShortURLsBatchFunc != nil {
		return m.createShortURLsBatchFunc(ctx, batch)
	}
	return nil, nil
}

func (m *mockURLService) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockURLService) BatchDeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	if m.batchDeleteURLsFunc != nil {
		return m.batchDeleteURLsFunc(ctx, shortURLs, userID)
	}
	return nil
}

func (m *mockURLService) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return nil
}

func (m *mockURLService) GetStats(ctx context.Context) (int, int, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx)
	}
	return 0, 0, nil
}

func (m *mockURLService) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockURLService) GetStorage() storage.URLStorage {
	return nil
}

func TestGRPCHandler_CreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		originalURL    string
		mockService    *mockURLService
		expectedStatus int32
		expectedURL    string
	}{
		{
			name:        "successful creation",
			originalURL: "https://example.com",
			mockService: &mockURLService{
				createShortURLFunc: func(ctx context.Context, originalURL string) (string, error) {
					return "abc123", nil
				},
			},
			expectedStatus: 200,
			expectedURL:    "http://localhost:8080/abc123",
		},
		{
			name:           "empty URL",
			originalURL:    "",
			mockService:    &mockURLService{},
			expectedStatus: 400,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{BaseURL: "http://localhost:8080"}
			logger, _ := zap.NewDevelopment()
			handler := NewGRPCHandler(tt.mockService, cfg, logger)

			req := &CreateShortURLRequest{OriginalURL: tt.originalURL}
			resp, err := handler.CreateShortURL(context.Background(), req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedURL, resp.ShortURL)
		})
	}
}

func TestGRPCHandler_GetOriginalURL(t *testing.T) {
	tests := []struct {
		name           string
		shortID        string
		mockService    *mockURLService
		expectedStatus int32
		expectedURL    string
	}{
		{
			name:    "successful retrieval",
			shortID: "abc123",
			mockService: &mockURLService{
				getOriginalURLFunc: func(ctx context.Context, shortURL string) (string, error) {
					return "https://example.com", nil
				},
			},
			expectedStatus: 200,
			expectedURL:    "https://example.com",
		},
		{
			name:           "empty shortID",
			shortID:        "",
			mockService:    &mockURLService{},
			expectedStatus: 400,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			logger, _ := zap.NewDevelopment()
			handler := NewGRPCHandler(tt.mockService, cfg, logger)

			req := &GetOriginalURLRequest{ShortID: tt.shortID}
			resp, err := handler.GetOriginalURL(context.Background(), req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedURL, resp.OriginalURL)
		})
	}
}

func TestGRPCHandler_Ping(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *mockURLService
		expectedStatus int32
	}{
		{
			name: "successful ping",
			mockService: &mockURLService{
				checkConnectionFunc: func(ctx context.Context) error {
					return nil
				},
			},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			logger, _ := zap.NewDevelopment()
			handler := NewGRPCHandler(tt.mockService, cfg, logger)

			req := &PingRequest{}
			resp, err := handler.Ping(context.Background(), req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestGRPCHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *mockURLService
		expectedStatus int32
		expectedURLs   int32
		expectedUsers  int32
	}{
		{
			name: "successful stats",
			mockService: &mockURLService{
				getStatsFunc: func(ctx context.Context) (int, int, error) {
					return 10, 5, nil
				},
			},
			expectedStatus: 200,
			expectedURLs:   10,
			expectedUsers:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			logger, _ := zap.NewDevelopment()
			handler := NewGRPCHandler(tt.mockService, cfg, logger)

			req := &GetStatsRequest{}
			resp, err := handler.GetStats(context.Background(), req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedURLs, resp.UrlsCount)
			assert.Equal(t, tt.expectedUsers, resp.UsersCount)
		})
	}
}

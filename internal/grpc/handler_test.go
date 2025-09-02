package grpc

import (
	"context"
	"testing"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MockURLService мок для URLService
type MockURLService struct {
	// Простая реализация без testify/mock для упрощения
	createShortURLFunc       func(ctx context.Context, originalURL string) (string, error)
	getOriginalURLFunc       func(ctx context.Context, shortID string) (string, error)
	createShortURLsBatchFunc func(ctx context.Context, entries []models.BatchRequestEntry) ([]models.BatchResponseEntry, error)
	getUserURLsFunc          func(ctx context.Context, userID string) ([]models.UserURL, error)
	batchDeleteURLsFunc      func(ctx context.Context, shortURLs []string, userID string) error
	checkConnectionFunc      func(ctx context.Context) error
	getStatsFunc             func(ctx context.Context) (int, int, error)
	closeFunc                func() error
}

func (m *MockURLService) CreateShortURL(ctx context.Context, originalURL string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(ctx, originalURL)
	}
	return "abc123", nil
}

func (m *MockURLService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(ctx, shortID)
	}
	return "https://example.com", nil
}

func (m *MockURLService) CreateShortURLsBatch(ctx context.Context, entries []models.BatchRequestEntry) ([]models.BatchResponseEntry, error) {
	if m.createShortURLsBatchFunc != nil {
		return m.createShortURLsBatchFunc(ctx, entries)
	}
	return []models.BatchResponseEntry{}, nil
}

func (m *MockURLService) GetUserURLs(ctx context.Context, userID string) ([]models.UserURL, error) {
	if m.getUserURLsFunc != nil {
		return m.getUserURLsFunc(ctx, userID)
	}
	return []models.UserURL{}, nil
}

func (m *MockURLService) BatchDeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	if m.batchDeleteURLsFunc != nil {
		return m.batchDeleteURLsFunc(ctx, shortURLs, userID)
	}
	return nil
}

func (m *MockURLService) CheckConnection(ctx context.Context) error {
	if m.checkConnectionFunc != nil {
		return m.checkConnectionFunc(ctx)
	}
	return nil
}

func (m *MockURLService) GetStats(ctx context.Context) (int, int, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx)
	}
	return 10, 5, nil
}

func (m *MockURLService) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockURLService) GetStorage() storage.URLStorage {
	return nil
}

func TestGRPCHandler_Execute(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	tests := []struct {
		name            string
		operationType   string
		expectedStatus  int32
		expectedSuccess bool
	}{
		{
			name:            "CreateShortURL operation",
			operationType:   OperationCreateShortURL,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "GetOriginalURL operation",
			operationType:   OperationGetOriginalURL,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "CreateShortURLsBatch operation",
			operationType:   OperationCreateShortURLsBatch,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "GetUserURLs operation",
			operationType:   OperationGetUserURLs,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "BatchDeleteURLs operation",
			operationType:   OperationBatchDeleteURLs,
			expectedStatus:  202,
			expectedSuccess: true,
		},
		{
			name:            "GetStats operation",
			operationType:   OperationGetStats,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "Unknown operation",
			operationType:   "unknown_operation",
			expectedStatus:  400,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockURLService)
			handler := NewGRPCHandler(mockService, cfg, logger)

			req := &ExecuteRequest{
				OperationType: tt.operationType,
				Payload:       &anypb.Any{},
			}

			resp, err := handler.Execute(context.Background(), req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedSuccess, resp.Success)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestGRPCHandler_Ping(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	tests := []struct {
		name            string
		mockError       error
		expectedStatus  int32
		expectedSuccess bool
	}{
		{
			name:            "Successful ping",
			mockError:       nil,
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name:            "Failed ping",
			mockError:       assert.AnError,
			expectedStatus:  503,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{
				checkConnectionFunc: func(ctx context.Context) error {
					return tt.mockError
				},
			}

			handler := NewGRPCHandler(mockService, cfg, logger)
			req := &emptypb.Empty{}

			resp, err := handler.Ping(context.Background(), req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedSuccess, resp.Success)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestGRPCHandler_OperationTypes(t *testing.T) {
	// Проверяем, что все константы типов операций определены
	assert.NotEmpty(t, OperationCreateShortURL)
	assert.NotEmpty(t, OperationGetOriginalURL)
	assert.NotEmpty(t, OperationCreateShortURLsBatch)
	assert.NotEmpty(t, OperationGetUserURLs)
	assert.NotEmpty(t, OperationBatchDeleteURLs)
	assert.NotEmpty(t, OperationGetStats)

	// Проверяем уникальность типов операций
	operationTypes := map[string]bool{
		OperationCreateShortURL:       true,
		OperationGetOriginalURL:       true,
		OperationCreateShortURLsBatch: true,
		OperationGetUserURLs:          true,
		OperationBatchDeleteURLs:      true,
		OperationGetStats:             true,
	}

	assert.Equal(t, 6, len(operationTypes), "All operation types should be unique")
}

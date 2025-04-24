package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		acceptEncoding     string
		contentEncoding    string
		contentType        string
		requestBody        string
		expectedStatusCode int
		expectedBody       string
		checkCompression   bool
		expectError        bool
	}{
		{
			name:               "Compress response when client supports gzip",
			acceptEncoding:     "gzip",
			contentType:        "text/plain",
			requestBody:        "test request",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "test response",
			checkCompression:   true,
			expectError:        false,
		},
		{
			name:               "Do not compress when client does not support gzip",
			acceptEncoding:     "",
			contentType:        "text/plain",
			requestBody:        "test request",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "test response",
			checkCompression:   false,
			expectError:        false,
		},
		{
			name:               "Decompress gzipped request",
			acceptEncoding:     "",
			contentEncoding:    "gzip",
			contentType:        "application/x-gzip",
			requestBody:        "test request",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "test response",
			checkCompression:   false,
			expectError:        false,
		},
		{
			name:               "Handle invalid gzip request",
			acceptEncoding:     "",
			contentEncoding:    "gzip",
			contentType:        "application/x-gzip",
			requestBody:        "invalid gzip data",
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "gzip: invalid header\n",
			checkCompression:   false,
			expectError:        true,
		},
		{
			name:               "Handle empty request body",
			acceptEncoding:     "",
			contentEncoding:    "gzip",
			contentType:        "application/x-gzip",
			requestBody:        "",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       "Empty request body\n",
			checkCompression:   false,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Проверяем, что Content-Type установлен правильно
				if tt.contentEncoding == "gzip" && r.Header.Get("Content-Type") != "text/plain" {
					t.Errorf("Expected Content-Type to be 'text/plain', got '%s'", r.Header.Get("Content-Type"))
				}

				// Читаем тело запроса только если не ожидаем ошибку
				if !tt.expectError {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("Error reading request body: %v", err)
						return
					}

					// Проверяем, что тело запроса распаковано правильно
					if tt.contentEncoding == "gzip" && string(body) != "test request" {
						t.Errorf("Expected request body to be 'test request', got '%s'", string(body))
					}
				}

				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.expectedStatusCode)
				w.Write([]byte(tt.expectedBody))
			})

			// Создаем тестовый запрос
			var body io.Reader
			if tt.contentEncoding == "gzip" {
				var buf strings.Builder
				gz := gzip.NewWriter(&buf)
				if _, err := gz.Write([]byte(tt.requestBody)); err != nil {
					t.Fatalf("Error writing gzipped data: %v", err)
				}
				if err := gz.Close(); err != nil {
					t.Fatalf("Error closing gzip writer: %v", err)
				}
				body = strings.NewReader(buf.String())
			} else {
				body = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			req.Header.Set("Content-Encoding", tt.contentEncoding)
			req.Header.Set("Content-Type", tt.contentType)

			// Создаем тестовый ответ
			w := httptest.NewRecorder()

			// Применяем middleware
			GzipMiddleware(handler).ServeHTTP(w, req)

			// Проверяем статус код
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			// Проверяем сжатие ответа
			if tt.checkCompression {
				if w.Header().Get("Content-Encoding") != "gzip" {
					t.Errorf("Expected Content-Encoding to be 'gzip', got '%s'", w.Header().Get("Content-Encoding"))
				}

				// Распаковываем ответ
				gz, err := gzip.NewReader(w.Body)
				if err != nil {
					t.Fatalf("Error creating gzip reader: %v", err)
				}
				defer gz.Close()

				body, err := io.ReadAll(gz)
				if err != nil {
					t.Fatalf("Error reading gzipped response: %v", err)
				}

				if string(body) != tt.expectedBody {
					t.Errorf("Expected response body to be '%s', got '%s'", tt.expectedBody, string(body))
				}
			} else {
				if w.Header().Get("Content-Encoding") == "gzip" {
					t.Error("Expected response not to be compressed")
				}

				if w.Body.String() != tt.expectedBody {
					t.Errorf("Expected response body to be '%s', got '%s'", tt.expectedBody, w.Body.String())
				}
			}
		})
	}
}

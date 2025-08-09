package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware обрабатывает сжатие и распаковку gzip
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, поддерживает ли клиент gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		// Проверяем, сжат ли входящий запрос
		contentEncoding := r.Header.Get("Content-Encoding")
		isGzipped := strings.Contains(contentEncoding, "gzip")

		// Если запрос сжат, распаковываем его
		if isGzipped {
			// Проверяем, что тело запроса не пустое
			if r.Body == nil {
				http.Error(w, "Empty request body", http.StatusBadRequest)
				return
			}

			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()
			r.Body = gz

			// Устанавливаем правильный Content-Type
			if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-gzip") {
				r.Header.Set("Content-Type", "text/plain")
			}
		}

		// Если клиент поддерживает gzip, сжимаем ответ
		if supportsGzip {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer func() {
				if err := gz.Close(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}()

			next.ServeHTTP(gzipResponseWriter{
				Writer:         gz,
				ResponseWriter: w,
			}, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// gzipResponseWriter оборачивает http.ResponseWriter для сжатия ответа
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Write записывает данные в сжатый поток
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// WriteHeader записывает код состояния HTTP ответа
func (w gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// Header возвращает HTTP заголовки ответа
func (w gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

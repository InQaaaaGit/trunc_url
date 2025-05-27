package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey используется как ключ для значений в контексте
type contextKey string

const (
	// UserIDKey используется как ключ для хранения ID пользователя в контексте
	UserIDKey  contextKey = "user_id"
	cookieName string     = "user_id"
	secretKey  string     = "your-secret-key" // В реальном приложении должен быть в конфигурации
)

// WithAuth middleware для аутентификации пользователя
func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем куку с токеном
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			// Если куки нет, возвращаем 401
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем токен
		claims := &models.UserClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims,
			func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secretKey), nil
			})

		if err != nil || !token.Valid {
			// Если токен невалиден, возвращаем 401
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Если токен валиден, добавляем userID в контекст
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateUserID генерирует уникальный идентификатор пользователя
func generateUserID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// В случае ошибки генерируем ID на основе времени
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// createToken создает JWT токен для пользователя
func createToken(userID string) (string, error) {
	claims := &models.UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

// GenerateUserID генерирует уникальный ID пользователя
func GenerateUserID() string {
	return uuid.New().String()
}

// SignUserID подписывает ID пользователя и возвращает строку "userID.signature"
func SignUserID(userID string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))
	return userID + "." + signature
}

// ValidateUserID проверяет подлинность подписанной строки userID
// Ожидает строку формата "userID.signature"
func ValidateUserID(signedValue string, secretKey string) (string, bool) {
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", false
	}
	userID := parts[0]
	signature := parts[1]

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return userID, hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GetUserIDFromSignedCookie извлекает userID из подписанной куки
// Этот вариант более правильный для ValidateUserID
// ValidateUserID должна принимать всю строку из куки
func GetUserIDFromSignedCookie(signedCookieValue string, secretKey string) (string, bool) {
	parts := strings.Split(signedCookieValue, ".")
	if len(parts) != 2 {
		return "", false
	}
	userID := parts[0]
	signature := parts[1]

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return userID, true
	}
	return "", false
}

// SignUserID должна теперь возвращать userID + "." + signature
func SignUserIDAndGenerateCookieValue(userID string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))
	return userID + "." + signature
}

package models

import "github.com/golang-jwt/jwt/v5"

// UserURL представляет собой запись о URL пользователя
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// UserClaims представляет собой данные, хранящиеся в JWT токене
type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

package storage

import "errors"

var (
	// ErrURLNotFound возвращается, когда URL не найден
	ErrURLNotFound = errors.New("url not found")
	// ErrOriginalURLConflict возвращается при попытке сохранить уже существующий оригинальный URL
	ErrOriginalURLConflict = errors.New("original URL already exists")
	// ErrInvalidURL возвращается при попытке сохранить некорректный URL
	ErrInvalidURL = errors.New("invalid URL format")
	// ErrURLAlreadyExists возвращается, когда URL уже существует
	ErrURLAlreadyExists = errors.New("url already exists")
)

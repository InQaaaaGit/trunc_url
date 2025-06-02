package storage

import "errors"

// ErrURLNotFound возвращается, когда URL не найден в хранилище
var ErrURLNotFound = errors.New("URL not found")

// ErrOriginalURLConflict возвращается, когда original_url уже существует
var ErrOriginalURLConflict = errors.New("original URL conflict")

// ErrURLDeleted возвращается, когда URL помечен как удаленный
var ErrURLDeleted = errors.New("URL is deleted")

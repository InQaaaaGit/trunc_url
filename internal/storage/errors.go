package storage

import "errors"

// ErrURLNotFound возвращается, когда URL не найден в хранилище
var ErrURLNotFound = errors.New("URL not found")

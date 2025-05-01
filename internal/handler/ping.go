package handler

import (
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// HandlePing обрабатывает запрос на проверку соединения с базой данных
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	// Проверяем, есть ли в сервисе хранилище с возможностью проверки соединения
	dbChecker, ok := h.service.GetStorage().(storage.DatabaseChecker)
	if !ok {
		// Если хранилище не реализует интерфейс DatabaseChecker, возвращаем ошибку
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Проверяем соединение с БД
	if err := dbChecker.CheckConnection(); err != nil {
		h.logger.Error("Ошибка подключения к БД", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Если соединение успешно, возвращаем 200 OK
	w.WriteHeader(http.StatusOK)
}

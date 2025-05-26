package handler

import (
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/storage"
	"go.uber.org/zap"
)

// HandlePing обрабатывает запрос на проверку соединения с базой данных
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем, есть ли в сервисе хранилище с возможностью проверки соединения
	dbChecker, ok := h.service.GetStorage().(storage.DatabaseChecker)
	if !ok {
		// Если хранилище не реализует интерфейс DatabaseChecker, возвращаем ошибку
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Проверяем соединение с БД
	if err := dbChecker.CheckConnection(r.Context()); err != nil {
		h.logger.Error("Ошибка подключения к БД", zap.Error(err))
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Если соединение успешно, возвращаем 200 OK
	w.WriteHeader(http.StatusOK)
}

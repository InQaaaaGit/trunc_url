package handler

import (
	"net/http"

	"go.uber.org/zap"
)

// HandlePing обрабатывает запрос на проверку соединения с базой данных
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем соединение через сервис
	if err := h.service.CheckConnection(r.Context()); err != nil {
		h.logger.Error("Ошибка подключения к хранилищу", zap.Error(err))
		http.Error(w, "Storage connection error", http.StatusInternalServerError)
		return
	}

	// Если соединение успешно, возвращаем 200 OK
	w.WriteHeader(http.StatusOK)
}

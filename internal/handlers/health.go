package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// HealthResponse представляет состояние сервиса в HTTP API
type HealthResponse struct {
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	LastCollection string `json:"last_collection,omitempty"`
}

// HealthHandler обрабатывает запросы к /health и возвращает состояние сервиса
func (h *Handler) HealthHandler(rw http.ResponseWriter, r *http.Request) {
	requestID, ok := r.Context().Value(requestIDKey).(string)
	if !ok {
		slog.Error("request_id отсутствует в context")
		http.Error(rw, "request_id отсутствует в context", http.StatusInternalServerError)
		return
	}

	logger := slog.With("request_id", requestID)

	logger.Info("получен запрос", "path", "/health")

	rw.Header().Set("Content-Type", "application/json")

	lastSuccess, lastError := h.system.GetHealth()

	if lastSuccess.IsZero() {
		logger.Error("при запросе по /health ещё нет успешных чтений метрик")
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  "ещё нет успешных чтений метрик",
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			logger.Error("не удалось записать ответ в JSON", "error", err)
		}

		return
	}

	if lastError != nil {
		logger.Error("ошибка при чтении метрик", "error", lastError)
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  fmt.Sprintf("ошибка при чтении метрик: %v", lastError),
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			logger.Error("не удалось записать ответ в JSON", "error", err)
		}
		return
	}

	if time.Since(lastSuccess) > time.Second*10 {
		logger.Error("последнее обновление данных случилось дольше 10 секунд назад")
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  "последнее обновление данных случилось дольше 10 секунд назад",
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			logger.Error("не удалось записать ответ в JSON", "error", err)
		}

		return
	}

	response := HealthResponse{
		Status:         "ok",
		LastCollection: lastSuccess.UTC().Format(time.RFC3339),
	}
	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		logger.Error("не удалось записать ответ в JSON", "error", err)
	}
}

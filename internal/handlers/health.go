package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	LastCollection string `json:"last_collection,omitempty"`
}

// HealthHandler отвечает на запросы по адресу /health и возвращает статус сервиса, а также
// ошибку или время последнего успешного сбора метрик в зависимости от статуса сервиса
func (h *Handler) HealthHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] получен запрос по адресу /health")

	rw.Header().Set("Content-Type", "application/json")

	lastSuccess, lastError := h.system.GetHealth()

	if lastSuccess.IsZero() {
		log.Print("[ERROR] при запросе по /health ещё нет успешных чтений метрик")
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  "ещё нет успешных чтений метрик",
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			log.Printf("[ERROR] не удалось записать ответ в JSON: %v", err)
		}

		return
	}

	if lastError != nil {
		log.Printf("[ERROR] ошибка при чтении метрик: %v", lastError)
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  fmt.Sprintf("ошибка при чтении метрик: %v", lastError),
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			log.Printf("[ERROR] не удалось записать ответ в JSON: %v", err)
		}
		return
	}

	if time.Since(lastSuccess) > time.Second*10 {
		log.Print("[ERROR] последнее обновление данных случилось дольше 10 секунд назад")
		rw.WriteHeader(http.StatusServiceUnavailable)

		response := HealthResponse{
			Status: "unhealthy",
			Error:  "последнее обновление данных случилось дольше 10 секунд назад",
		}

		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			log.Printf("[ERROR] не удалось записать ответ в JSON: %v", err)
		}

		return
	}

	response := HealthResponse{
		Status:         "ok",
		LastCollection: lastSuccess.UTC().Format(time.RFC3339),
	}
	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		log.Printf("[ERROR] не удалось записать ответ в JSON: %v", err)
	}
}

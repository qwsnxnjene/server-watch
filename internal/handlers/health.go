package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

//TODO: сделать общую структуру для ответа, поменять сериализацию JSON

func (h *Handler) HealthHandler(rw http.ResponseWriter, r *http.Request) {
	lastSuccess, lastError := h.system.GetHealth()

	if lastSuccess.IsZero() {
		log.Print("[ERROR] при запросе по /health ещё нет успешных чтений метрик")
		rw.WriteHeader(http.StatusServiceUnavailable)

		resp := struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{Status: "unhealthy",
			Error: "еще нет успешных чтений метрик"}

		respJSON, _ := json.Marshal(resp)
		rw.Write(respJSON)

		return
	}

	if lastError != nil {
		log.Printf("[ERROR] ошибка при чтении метрик: %v", lastError)
		rw.WriteHeader(http.StatusServiceUnavailable)

		resp := struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{Status: "unhealthy",
			Error: fmt.Sprintf("ошибка при чтении метрик: %v", lastError)}

		respJSON, _ := json.Marshal(resp)
		rw.Write(respJSON)
		return
	}

	if time.Since(lastSuccess) > time.Second*10 {
		log.Print("[ERROR] последнее обновление данных случилось дольше 10 секунд назад")
		rw.WriteHeader(http.StatusServiceUnavailable)

		resp := struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{Status: "unhealthy",
			Error: "последнее обновление данных случилось дольше 10 секунд назад"}

		respJSON, _ := json.Marshal(resp)
		rw.Write(respJSON)

		return
	}

	resp := struct {
		Status         string `json:"status"`
		LastCollection string `json:"last_collection"`
	}{Status: "ok",
		LastCollection: lastSuccess.Format("2006-01-02T15:04:05Z")}
	respJSON, _ := json.Marshal(resp)
	rw.Write(respJSON)
}

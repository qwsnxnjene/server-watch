package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"server-watch/internal/system"
	"time"
)

type AlertResponse struct {
	ID         int64            `json:"id"`
	Type       system.AlertType `json:"alert_type"`
	Timestamp  time.Time        `json:"ts"`
	Threshold  float64          `json:"threshold"`
	Resolved   bool             `json:"resolved"`
	ResolvedAt *time.Time       `json:"resolvedAt"`
	Value      float64          `json:"value"`
}

type AlertsResponse struct {
	Alerts []AlertResponse `json:"alerts"`
}

// AlertsHandler отвечает за запросы по адресу /alerts и возвращает список всех/только активных алертов
// в зависимости от значения параметра active_only
func (h *Handler) AlertsHandler(rw http.ResponseWriter, r *http.Request) {
	slog.Info("получен запрос", "path", "/alerts")

	activeOnly := r.URL.Query().Get("active_only")
	if activeOnly == "" {
		activeOnly = "false"
	}

	var parsedActiveOnly bool
	if activeOnly == "true" {
		parsedActiveOnly = true
	} else if activeOnly == "false" {
		parsedActiveOnly = false
	} else {
		slog.Warn("некорректное значение параметра active_only", "active_only", activeOnly)
		http.Error(rw, "некорректное значение параметра active_only", http.StatusBadRequest)
		return
	}

	alerts, err := h.system.GetAlerts(parsedActiveOnly)
	if err != nil {
		slog.Error("не удалось получить алерты", "error", err)
		http.Error(rw, "ошибка получения списка алертов", http.StatusInternalServerError)
		return
	}

	alertsResp := make([]AlertResponse, 0, len(alerts))
	for _, alert := range alerts {
		alertResp := AlertResponse{
			ID:         alert.ID,
			Type:       alert.Type,
			Timestamp:  alert.Timestamp,
			Threshold:  alert.Threshold,
			Resolved:   alert.Resolved,
			ResolvedAt: alert.ResolvedAt,
			Value:      alert.Value,
		}

		alertsResp = append(alertsResp, alertResp)
	}

	response := AlertsResponse{Alerts: alertsResp}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		slog.Error("не удалось сериализовать список алертов", "error", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

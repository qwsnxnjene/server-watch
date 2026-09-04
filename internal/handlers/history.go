package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type HistoryResponse struct {
	Metrics []MetricsResponse `json:"metrics"`
}

// HistoryHandler отвечает на запросы по адресу /history и возвращает список измерений метрик
// в заданный промежуток времени от from до to
func (h *Handler) HistoryHandler(rw http.ResponseWriter, r *http.Request) {
	slog.Info("получен запрос", "path", "/history")

	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	}
	if to == "" {
		to = time.Now().UTC().Format(time.RFC3339)
	}

	parsedFrom, err := time.Parse(time.RFC3339, from)
	if err != nil {
		slog.Warn("некорректное значение параметра from в запросе", "from", from)
		http.Error(rw, "некорректное значение параметра from", http.StatusBadRequest)
		return
	}

	parsedTo, err := time.Parse(time.RFC3339, to)
	if err != nil {
		slog.Warn("некорректное значение параметра to в запросе", "to", to)
		http.Error(rw, "некорректное значение параметра to", http.StatusBadRequest)
		return
	}

	if parsedFrom.After(parsedTo) {
		slog.Warn("параметр from не может быть позже параметра to", "from", from, "to", to)
		http.Error(rw, "параметр from не может быть позже параметра to", http.StatusBadRequest)
		return
	}

	metrics, err := h.system.GetHistory(parsedFrom, parsedTo)
	if err != nil {
		slog.Error("ошибка получения истории измерения метрик", "error", err)
		http.Error(rw, "не удалось получить историю измерения метрик", http.StatusInternalServerError)
		return
	}

	metricsResp := make([]MetricsResponse, 0, len(metrics))
	for _, metric := range metrics {
		metricResp := MetricsResponse{
			CPUPercent:  metric.CPUUsage,
			MemPercent:  metric.MemUsage,
			MemUsedMB:   metric.MemUsedMB,
			MemTotalMB:  metric.MemTotalMB,
			DiskPercent: metric.DiskUsage,
			DiskUsedGB:  metric.DiskUsed,
			DiskTotalGB: metric.DiskTotal,
		}

		metricsResp = append(metricsResp, metricResp)
	}

	response := HistoryResponse{Metrics: metricsResp}

	rw.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		slog.Error("не удалось сериализовать историю метрик", "error", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	logger2 "server-watch/internal/logger"
	"time"
)

// HistoryResponse представляет историю измерений метрик в HTTP API
type HistoryResponse struct {
	Metrics []MetricsResponse `json:"metrics"`
}

// HistoryHandler обрабатывает запросы к /history и возвращает историю измерений метрик
func (h *Handler) HistoryHandler(rw http.ResponseWriter, r *http.Request) {
	requestID, ok := r.Context().Value(requestIDKey).(string)
	if !ok {
		slog.Error("request_id отсутствует в context")
		http.Error(rw, "request_id отсутствует в context", http.StatusInternalServerError)
		return
	}

	logger := slog.With("request_id", requestID)
	ctx := logger2.WithLogger(r.Context(), logger)
	r = r.WithContext(ctx)

	logger.Info("получен запрос", "path", "/history")

	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	}
	if to == "" {
		to = time.Now().UTC().Format(time.RFC3339)
	}

	parsedFrom, err := time.Parse(time.RFC3339, from)
	if err != nil {
		logger.Warn("некорректное значение параметра from в запросе", "from", from)
		http.Error(rw, "некорректное значение параметра from", http.StatusBadRequest)
		return
	}

	parsedTo, err := time.Parse(time.RFC3339, to)
	if err != nil {
		logger.Warn("некорректное значение параметра to в запросе", "to", to)
		http.Error(rw, "некорректное значение параметра to", http.StatusBadRequest)
		return
	}

	if parsedFrom.After(parsedTo) {
		logger.Warn("параметр from не может быть позже параметра to", "from", from, "to", to)
		http.Error(rw, "параметр from не может быть позже параметра to", http.StatusBadRequest)
		return
	}

	metrics, err := h.system.GetHistory(r.Context(), parsedFrom, parsedTo)
	if err != nil {
		logger.Error("ошибка получения истории измерения метрик", "error", err)
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
		logger.Error("не удалось сериализовать историю метрик", "error", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

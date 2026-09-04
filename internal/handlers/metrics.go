package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strings"
)

type MetricsResponse struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	MemUsedMB   float64 `json:"mem_used_mb"`
	MemTotalMB  float64 `json:"mem_total_mb"`
	DiskPercent float64 `json:"disk_percent"`
	DiskUsedGB  float64 `json:"disk_used_gb"`
	DiskTotalGB float64 `json:"disk_total_gb"`
}

// MetricsHandler отвечает на запросы по адресу /metrics и возвращает
// актуальные на данный момент метрики
func (h *Handler) MetricsHandler(rw http.ResponseWriter, r *http.Request) {
	slog.Info("получен запрос", "path", "/metrics")

	// перенаправляем на Prometheus
	if ok, err := acceptsPrometheus(r); err == nil {
		if ok {
			h.prometheusHandler.ServeHTTP(rw, r)
			return
		}
	} else {
		slog.Error("не удалось спарсить Accept", "error", err)
		http.Error(rw, "не удалось спарсить Accept", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	metrics := h.system.GetMetrics()

	response := MetricsResponse{
		CPUPercent:  metrics.CPUUsage,
		MemPercent:  metrics.MemUsage,
		MemUsedMB:   metrics.MemUsedMB,
		MemTotalMB:  metrics.MemTotalMB,
		DiskPercent: metrics.DiskUsage,
		DiskUsedGB:  metrics.DiskUsed,
		DiskTotalGB: metrics.DiskTotal,
	}

	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		slog.Error("не удалось сериализовать метрики", "error", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func acceptsPrometheus(r *http.Request) (bool, error) {
	for _, accept := range strings.Split(r.Header.Get("Accept"), ",") {
		mediaType, params, err := mime.ParseMediaType(accept)
		if err != nil {
			return false, fmt.Errorf("не удалось спарсить Accept: %w", err)
		}

		if params["q"] == "0" {
			continue
		}

		if mediaType == "text/plain" || mediaType == "application/openmetrics-text" {
			return true, nil
		}
	}

	return false, nil
}

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
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
	log.Println("[INFO] получен запрос по адресу /metrics")
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
		log.Printf("[ERROR] не удалось сериализовать метрики: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

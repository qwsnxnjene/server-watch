package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "application/json")

	metrics := h.system.GetMetrics()

	resp, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("[ERROR] не удалось сериализовать метрики: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(resp)
}

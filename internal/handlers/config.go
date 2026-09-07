package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"server-watch/internal/config"
	"server-watch/internal/system"
)

// ConfigHandler отвечает на запросы по адресу /config и обновляет конфигурацию сервиса
// в зависимости от переданных параметров
func (h *Handler) ConfigHandler(rw http.ResponseWriter, r *http.Request) {
	slog.Info("получен запрос", "path", "/config")

	var updatedCfg system.ConfigUpdate

	if err := json.NewDecoder(r.Body).Decode(&updatedCfg); err != nil {
		http.Error(rw, "некорректный JSON", http.StatusBadRequest)
		return
	}

	err := h.system.UpdateConfig(updatedCfg)
	if err != nil {
		if errors.Is(err, config.ErrInvalidConfig) {
			http.Error(rw, "невалидная конфигурация", http.StatusBadRequest)
			return
		}
		http.Error(rw,
			"не удалось обновить конфигурацию",
			http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	cfg := h.system.GetConfig()
	err = json.NewEncoder(rw).Encode(cfg)
	if err != nil {
		http.Error(rw, "не удалось закодировать ответ", http.StatusInternalServerError)
	}
}

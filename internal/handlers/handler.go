package handlers

import (
	"net/http"
	"server-watch/internal/config"
	"server-watch/internal/system"
	"server-watch/internal/system/model"
	"time"
)

// System - интерфейс для уровня бизнес-логики
type System interface {
	GetMetrics() system.Metrics
	GetHistory(from, to time.Time) ([]system.Metrics, error)
	GetAlerts(activeOnly bool) ([]model.Alert, error)
	GetHealth() (time.Time, error)

	UpdateConfig(updatedCfg system.ConfigUpdate) error
	GetConfig() config.Config
}

type Handler struct {
	system            System
	prometheusHandler http.Handler
}

func NewHandler(sys System, promHandler http.Handler) *Handler {
	return &Handler{
		system:            sys,
		prometheusHandler: promHandler,
	}
}

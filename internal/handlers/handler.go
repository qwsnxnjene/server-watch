package handlers

import (
	"net/http"
	"server-watch/internal/system"
	"time"
)

// System - интерфейс для уровня бизнес-логики
type System interface {
	GetMetrics() system.Metrics
	GetHistory(from, to time.Time) ([]system.Metrics, error)
	GetAlerts(activeOnly bool) ([]system.Alert, error)
	GetHealth() (time.Time, error)
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

package handlers

import (
	"context"
	"net/http"
	"server-watch/internal/config"
	"server-watch/internal/system"
	"server-watch/internal/system/model"
	"time"
)

// System определяет интерфейс бизнес-логики,
// необходимый HTTP-обработчикам
type System interface {
	GetMetrics(ctx context.Context) model.Metrics
	GetHistory(ctx context.Context, from, to time.Time) ([]model.Metrics, error)
	GetAlerts(ctx context.Context, activeOnly bool) ([]model.Alert, error)
	GetHealth() (time.Time, error)

	UpdateConfig(updatedCfg system.ConfigUpdate) error
	GetConfig() config.Config
}

type Handler struct {
	system            System
	prometheusHandler http.Handler
}

// NewHandler создаёт HTTP-обработчик и связывает его с бизнес-логикой
func NewHandler(sys System, promHandler http.Handler) *Handler {
	return &Handler{
		system:            sys,
		prometheusHandler: promHandler,
	}
}

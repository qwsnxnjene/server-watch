package system

import (
	"server-watch/internal/system/model"
	"time"
)

// Repository - интерфейс для хранения данных
type Repository interface {
	SaveMetrics(metrics Metrics) error
	SaveAlert(alert model.Alert) (int64, error)
	GetMetrics(from time.Time, to time.Time) ([]Metrics, error)
	GetAlerts(activeOnly bool) ([]model.Alert, error)
	ResolveAlert(id int64, resolvedAt time.Time) error
	GetActiveAlert(alertType model.AlertType) (*model.Alert, error)
}

package system

import (
	"time"
)

// Repository - интерфейс для хранения данных
type Repository interface {
	SaveMetrics(metrics Metrics) error
	SaveAlert(alert Alert) (int64, error)
	GetMetrics(from time.Time, to time.Time) ([]Metrics, error)
	GetAlerts(activeOnly bool) ([]Alert, error)
	ResolveAlert(id int64, resolvedAt time.Time) error
	GetActiveAlert(alertType AlertType) (*Alert, error)
}

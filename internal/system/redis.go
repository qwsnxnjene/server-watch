package system

import "errors"

var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

type AlertStateStore interface {
	IncrementCount(alertType AlertType) (int64, error)
	ResetCount(alertType AlertType) error

	IsActive(alertType AlertType) (bool, error)
	SetActive(alertType AlertType, active bool) error
}

package system

import (
	"errors"
	"server-watch/internal/system/model"
)

var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

type AlertStateStore interface {
	IncrementCount(alertType model.AlertType, condition model.AlertCondition) (int64, error)

	IsActive(alertType model.AlertType) (bool, error)
	SetActive(alertType model.AlertType, active bool) error
}

type AlertStateSnapshotStore interface {
	GetState(alertType model.AlertType) (model.AlertState, error)
	SetState(alertType model.AlertType, state model.AlertState) error
}

type AlertStateBackend interface {
	AlertStateStore
	AlertStateSnapshotStore
}

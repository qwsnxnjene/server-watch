package system

import "errors"

var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

type AlertCondition string

const (
	ConditionNormal AlertCondition = "normal"
	ConditionHigh   AlertCondition = "high"
)

type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

type AlertStateStore interface {
	IncrementCount(alertType AlertType, condition AlertCondition) (int64, error)

	IsActive(alertType AlertType) (bool, error)
	SetActive(alertType AlertType, active bool) error
}

type AlertStateSnapshotStore interface {
	GetState(alertType AlertType) (AlertState, error)
	SetState(alertType AlertType, state AlertState) error
}

type AlertStateBackend interface {
	AlertStateStore
	AlertStateSnapshotStore
}

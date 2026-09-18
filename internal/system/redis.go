package system

import (
	"errors"
	"server-watch/internal/system/model"
)

// ErrCacheMiss означает, что метрики отсутствуют в кэше
var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

// MetricsCache определяет операции сохранения и получения системных метрик из кэша
type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

// AlertStateStore управляет счётчиком последовательных условий
// и активностью алертов.
type AlertStateStore interface {
	IncrementCount(alertType model.AlertType, condition model.AlertCondition) (int64, error)

	IsActive(alertType model.AlertType) (bool, error)
	SetActive(alertType model.AlertType, active bool) error
}

// AlertStateSnapshotStore определяет операции чтения и сохранения
// полного состояния алерта
type AlertStateSnapshotStore interface {
	GetState(alertType model.AlertType) (model.AlertState, error)
	SetState(alertType model.AlertType, state model.AlertState) error
}

// AlertStateBackend объединяет операции над текущим состоянием алерта
// и его полным снимком
type AlertStateBackend interface {
	AlertStateStore
	AlertStateSnapshotStore
}

package system

import (
	"context"
	"errors"
	"server-watch/internal/system/model"
)

// ErrCacheMiss означает, что метрики отсутствуют в кэше
var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

// MetricsCache определяет операции сохранения и получения системных метрик из кэша
type MetricsCache interface {
	SetMetrics(ctx context.Context, metrics model.Metrics) error
	GetMetrics(ctx context.Context) (model.Metrics, error)
}

// AlertStateStore управляет счётчиком последовательных условий
// и активностью алертов.
type AlertStateStore interface {
	IncrementCount(ctx context.Context, alertType model.AlertType, condition model.AlertCondition) (int64, error)

	IsActive(ctx context.Context, alertType model.AlertType) (bool, error)
	SetActive(ctx context.Context, alertType model.AlertType, active bool) error
}

// AlertStateSnapshotStore определяет операции чтения и сохранения
// полного состояния алерта
type AlertStateSnapshotStore interface {
	GetState(ctx context.Context, alertType model.AlertType) (model.AlertState, error)
	SetState(ctx context.Context, alertType model.AlertType, state model.AlertState) error
}

// AlertStateBackend объединяет операции над текущим состоянием алерта
// и его полным снимком
type AlertStateBackend interface {
	AlertStateStore
	AlertStateSnapshotStore
}

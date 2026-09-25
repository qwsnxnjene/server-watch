package system

import (
	"context"
	"server-watch/internal/system/model"
	"time"
)

// Repository определяет хранилище данных, необходимое системному слою
type Repository interface {
	SaveMetrics(ctx context.Context, metrics model.Metrics) error
	SaveAlert(ctx context.Context, alert model.Alert) (int64, error)
	GetMetrics(ctx context.Context, from time.Time, to time.Time) ([]model.Metrics, error)
	GetAlerts(ctx context.Context, activeOnly bool) ([]model.Alert, error)
	ResolveAlert(ctx context.Context, id int64, resolvedAt time.Time) error
	GetActiveAlert(ctx context.Context, alertType model.AlertType) (*model.Alert, error)
}

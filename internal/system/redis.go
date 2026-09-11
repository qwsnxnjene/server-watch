package system

import "errors"

var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

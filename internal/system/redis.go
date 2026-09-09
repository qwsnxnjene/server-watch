package system

type MetricsCache interface {
	SetMetrics(metrics Metrics) error
	GetMetrics() (Metrics, error)
}

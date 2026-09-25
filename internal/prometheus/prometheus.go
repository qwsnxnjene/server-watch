package prometheus

import (
	"fmt"
	"net/http"
	"server-watch/internal/system/model"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var HTTPLatency = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Время обработки HTTP-запросов",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 13),
	},
	[]string{"path", "method", "status"},
)

var cpuPercent = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "cpu_percent",
	Help: "Текущее использование CPU в процентах",
})

var memUsedMb = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "mem_used_mb",
	Help: "Текущее количество используемой памяти в МБ",
})

var memTotalMb = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "mem_total_mb",
	Help: "Общее количество всей памяти в МБ",
})

var diskUsedGb = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "disk_used_gb",
	Help: "Текущее количество используемого места на диске в ГБ",
})

var diskTotalGb = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "disk_total_gb",
	Help: "Общее количество всего места на диске в ГБ",
})

var AlertsActiveTotal = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "alerts_active_total",
	Help: "Количество активных алертов на данный момент",
})

var AlertsTotal = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "alerts_total",
	Help: "Общее количество алертов с момента запуска сервиса",
})

var registry = prometheus.NewRegistry()

// RegisterPrometheusMetrics регистрирует метрики сервиса в Prometheus registry
func RegisterPrometheusMetrics() error {
	err := registry.Register(cpuPercent)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику cpu_percent: %w", err)
	}

	err = registry.Register(memUsedMb)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику mem_used_mb: %w", err)
	}

	err = registry.Register(memTotalMb)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику mem_total_mb: %w", err)
	}

	err = registry.Register(diskUsedGb)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику disk_used_gb: %w", err)
	}

	err = registry.Register(diskTotalGb)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику disk_total_gb: %w", err)
	}

	err = registry.Register(AlertsActiveTotal)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику alerts_active_total: %w", err)
	}

	err = registry.Register(AlertsTotal)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику alerts_total: %w", err)
	}

	err = registry.Register(HTTPLatency)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать HTTP latency: %w", err)
	}

	return nil
}

func UpdatePrometheusMetrics(metrics model.Metrics) {
	cpuPercent.Set(metrics.CPUUsage)
	memUsedMb.Set(metrics.MemUsedMB)
	memTotalMb.Set(metrics.MemTotalMB)
	diskUsedGb.Set(metrics.DiskUsed)
	diskTotalGb.Set(metrics.DiskTotal)
}

// PrometheusHandler возвращает HTTP-обработчик для сбора метрик Prometheus
func PrometheusHandler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

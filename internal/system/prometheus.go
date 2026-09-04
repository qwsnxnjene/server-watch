package system

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
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

var alertsActiveTotal = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "alerts_active_total",
	Help: "Количество активных алертов на данный момент",
})

var alertsTotal = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "alerts_total",
	Help: "Общее количество алертов с момента запуска сервиса",
})

var registry = prometheus.NewRegistry()

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

	err = registry.Register(alertsActiveTotal)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику alerts_active_total: %w", err)
	}

	err = registry.Register(alertsTotal)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать в Prometheus метрику alerts_total: %w", err)
	}

	return nil
}

func updatePrometheusMetrics(metrics Metrics) {
	cpuPercent.Set(metrics.CPUUsage)
	memUsedMb.Set(metrics.MemUsedMB)
	memTotalMb.Set(metrics.MemTotalMB)
	diskUsedGb.Set(metrics.DiskUsed)
	diskTotalGb.Set(metrics.DiskTotal)
}

func PrometheusHandler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

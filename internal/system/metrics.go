package system

import (
	"fmt"
	"log/slog"
	"time"
)

type Metrics struct {
	CPUUsage   float64
	MemUsage   float64
	MemUsedMB  float64
	MemTotalMB float64
	DiskUsage  float64
	DiskUsed   float64
	DiskTotal  float64
	Timestamp  time.Time
}

// GetMetrics возвращает копию актуальных метрик
func (s *System) GetMetrics() Metrics {
	if s.cache != nil {
		if metrics, err := s.cache.GetMetrics(); err == nil {
			return metrics
		} else {
			slog.Warn("не удалось получить метрики из кэша", "error", err)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.metrics
}

// GetHistory возвращает историю измерений метрик в заданных временных рамках
func (s *System) GetHistory(from, to time.Time) ([]Metrics, error) {
	metrics, err := s.repository.GetMetrics(from, to)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список измерений метрик: %w", err)
	}

	return metrics, nil
}

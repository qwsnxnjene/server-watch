package system

import (
	"fmt"
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

// GetMetrics возвращает копию метрик
func (s *System) GetMetrics() Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.metrics
}

//TODO: тесты для GetHistory

// GetHistory возвращает историю измерений метрик в заданных временных рамках
func (s *System) GetHistory(from, to time.Time) ([]Metrics, error) {
	metrics, err := s.repository.GetMetrics(from, to)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список измерений метрик: %w", err)
	}

	return metrics, nil
}

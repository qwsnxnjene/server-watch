package system

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"server-watch/internal/logger"
	"time"
)

// Metrics содержит значения системных метрик и дату проведенного замера
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
func (s *System) GetMetrics(ctx context.Context) Metrics {
	loggerFromContext, ok := logger.FromContext(ctx)
	if !ok {
		loggerFromContext = slog.Default()
	}

	if s.cache != nil {
		if metrics, err := s.cache.GetMetrics(ctx); err == nil {
			return metrics
		} else {
			if !errors.Is(err, context.Canceled) {
				loggerFromContext.Warn("не удалось получить метрики из кэша", "error", err)
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.metrics
}

// GetHistory возвращает историю измерений метрик в заданных временных рамках
func (s *System) GetHistory(ctx context.Context, from, to time.Time) ([]Metrics, error) {
	metrics, err := s.repository.GetMetrics(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список измерений метрик: %w", err)
	}

	return metrics, nil
}

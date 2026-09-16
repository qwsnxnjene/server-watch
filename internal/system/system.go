package system

import (
	"fmt"
	"log/slog"
	"server-watch/internal/config"
	"server-watch/internal/system/data"
	"sync"
	"time"
)

const (
	HighCPUThreshold = 80.0
	HighMemThreshold = 90.0

	AlertTriggerCount = 3
	AlertResolveCount = 3
)

// System - системный слой, ответственный за бизнес-логику
type System struct {
	mu          sync.RWMutex
	metrics     Metrics
	lastSuccess time.Time
	lastError   error
	repository  Repository

	config     config.Config
	configPath string

	alertState AlertStateStore

	cache MetricsCache
}

func NewSystem(
	repository Repository,
	alertState AlertStateStore,
	cfg config.Config,
	path string,
	cache MetricsCache) *System {
	return &System{
		repository: repository,
		alertState: alertState,
		config:     cfg,
		configPath: path,
		cache:      cache,
	}
}

// CollectMetrics с помощью вспомогательных функций собирает свежие данные с ОС
func (s *System) CollectMetrics() error {
	usage, err := data.GetCPUUsage()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о загрузке CPU: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalMem, usedMem, memUsage, err := data.ReadMemoryStats()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о памяти: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalDisk, usedDisk, diskUsage, err := data.GetDiskStats()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о диске: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	now := time.Now()

	metrics := Metrics{
		CPUUsage:   usage,
		MemUsage:   memUsage,
		MemUsedMB:  usedMem,
		MemTotalMB: totalMem,
		DiskUsage:  diskUsage,
		DiskUsed:   usedDisk,
		DiskTotal:  totalDisk,
		Timestamp:  now,
	}

	if err := s.repository.SaveMetrics(metrics); err != nil {
		errToReturn := fmt.Errorf("не удалось сохранить метрики в БД: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	// обновляем актуальные измерения метрик на данный момент
	s.mu.Lock()
	s.metrics = metrics
	s.lastError = nil
	s.lastSuccess = metrics.Timestamp
	s.mu.Unlock()

	updatePrometheusMetrics(metrics)

	update, err := s.updateAlerts(metrics)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обновить состояния алертов: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	err = s.processAlerts(metrics, update)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обработать алерты: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	if s.cache != nil {
		err = s.cache.SetMetrics(metrics)
		if err != nil {
			slog.Warn("не удалось сохранить метрики в кэш", "error", err)
		}
	}

	slog.Info("метрики успешно обновлены")

	return nil
}

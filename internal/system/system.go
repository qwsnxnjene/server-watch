package system

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	HighCPUThreshold = 80.0
	HighMemThreshold = 90.0

	AlertTriggerCount = 3
	AlertResolveCount = 3
)

type System struct {
	mu          sync.RWMutex
	metrics     Metrics
	lastSuccess time.Time
	lastError   error
	repository  Repository

	AlertCPU AlertState
	AlertMem AlertState
}

func NewSystem(repository Repository) *System {
	return &System{repository: repository}
}

// CollectMetrics с помощью вспомогательных функций собирает свежие данные с ОС
func (s *System) CollectMetrics() error {
	usage, err := getCPUUsage()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о загрузке CPU: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalMem, usedMem, memUsage, err := readMemoryStats()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о памяти: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalDisk, usedDisk, diskUsage, err := getDiskStats()
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

	s.mu.Lock()
	s.metrics = metrics
	s.lastError = nil
	s.lastSuccess = metrics.Timestamp
	s.mu.Unlock()

	s.updateAlerts(metrics)

	err = s.processAlerts(metrics)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обработать алерты: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	log.Println("[INFO] метрики успешно обновлены")

	return nil
}

func (s *System) updateAlerts(metrics Metrics) {
	s.AlertCPU.Record(metrics.CPUUsage, HighCPUThreshold)
	s.AlertMem.Record(metrics.MemUsage, HighMemThreshold)
}

func (s *System) processAlerts(metrics Metrics) error {
	if s.AlertCPU.HighThresholdReached() {
		err := s.createAlertIfNeeded(AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if s.AlertCPU.ResolveThresholdReached() {
		err := s.resolveAlertIfNeeded(AlertTypeHighCPU)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	//с памятью точно также
	if s.AlertMem.HighThresholdReached() {
		err := s.createAlertIfNeeded(AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if s.AlertMem.ResolveThresholdReached() {
		err := s.resolveAlertIfNeeded(AlertTypeHighMem)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	return nil
}

func (s *System) createAlertIfNeeded(alertType AlertType, value float64, threshold float64) error {
	alert, err := s.repository.GetActiveAlert(alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert == nil {
		alertToSave := Alert{
			Type:       alertType,
			Timestamp:  time.Now(),
			Threshold:  threshold,
			Resolved:   false,
			ResolvedAt: nil,
			Value:      value,
		}

		_, err = s.repository.SaveAlert(alertToSave)
		if err != nil {
			return fmt.Errorf("не удалось сохранить новый алерт типа %v: %w", alertType, err)
		}
	}

	return nil
}

func (s *System) resolveAlertIfNeeded(alertType AlertType) error {
	alert, err := s.repository.GetActiveAlert(alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert != nil {
		err = s.repository.ResolveAlert(alert.ID, time.Now())
		if err != nil {
			return fmt.Errorf("не удалось зарезолвить алерт типа %v: %w", alertType, err)
		}
	}

	return nil
}

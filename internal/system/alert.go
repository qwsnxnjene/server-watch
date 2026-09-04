package system

import (
	"fmt"
	"log/slog"
	"time"
)

type AlertType string

const (
	AlertTypeHighCPU AlertType = "HIGH_CPU"
	AlertTypeHighMem AlertType = "HIGH_MEM"
)

type Alert struct {
	ID         int64
	Type       AlertType
	Timestamp  time.Time
	Threshold  float64
	Resolved   bool
	ResolvedAt *time.Time
	Value      float64
}

// GetAlerts возвращает список алертов с возможностью выбрать только активные с помощью флага activeOnly
func (s *System) GetAlerts(activeOnly bool) ([]Alert, error) {
	alerts, err := s.repository.GetAlerts(activeOnly)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список алертов: %w", err)
	}

	return alerts, nil
}

type AlertState struct {
	consecutiveHigh   int
	consecutiveNormal int
}

// Record обновляет счетчики алерта в зависимости от значения и порога
func (a *AlertState) Record(value float64, threshold float64) {
	if value > threshold {
		a.consecutiveHigh++
		a.consecutiveNormal = 0
	} else {
		a.consecutiveNormal++
		a.consecutiveHigh = 0
	}
}

// HighThresholdReached сообщает о том, превышено ли количество подряд идущих измерений выше нормы
func (a *AlertState) HighThresholdReached() bool {
	return a.consecutiveHigh >= AlertTriggerCount
}

// ResolveThresholdReached сообщает о том, превышено ли количество подряд идущих измерений в рамках нормы
func (a *AlertState) ResolveThresholdReached() bool {
	return a.consecutiveNormal >= AlertResolveCount
}

func (s *System) updateAlerts(metrics Metrics) {
	if metrics.CPUUsage > HighCPUThreshold {
		slog.Warn(
			"превышен порог CPU",
			"value", metrics.CPUUsage,
			"threshold", HighCPUThreshold,
		)
	}

	if metrics.MemUsage > HighMemThreshold {
		slog.Warn(
			"превышен порог памяти",
			"value", metrics.MemUsage,
			"threshold", HighMemThreshold,
		)
	}

	// обновляем данные об алертах для процессора и памяти
	s.AlertCPU.Record(metrics.CPUUsage, HighCPUThreshold)
	s.AlertMem.Record(metrics.MemUsage, HighMemThreshold)
}

func (s *System) processAlerts(metrics Metrics) error {
	// проверяем четыре сценария, по 2 на процессор и память (создание и резолв алерта)

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
	// если активного алерта на данный момент нет, то сохраняем новый, иначе не делаем ничего

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
		slog.Info("создан новый алерт!",
			"type", alertToSave.Type,
			"threshold", alertToSave.Threshold,
			"value", alertToSave.Value)
	}

	return nil
}

func (s *System) resolveAlertIfNeeded(alertType AlertType) error {
	// пробуем зарезолвить алерт по id

	alert, err := s.repository.GetActiveAlert(alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert != nil {
		err = s.repository.ResolveAlert(alert.ID, time.Now())
		if err != nil {
			return fmt.Errorf("не удалось зарезолвить алерт типа %v: %w", alertType, err)
		}
		slog.Info("зарезолвлен алерт!",
			"type", alert.Type,
			"value", alert.Value)
	}

	return nil
}

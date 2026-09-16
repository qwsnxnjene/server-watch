package system

import (
	"fmt"
	"log/slog"
	"time"
)

// AlertType - тип алерта (CPU или MEM)
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

type alertUpdate struct {
	CPUCount     int64
	CPUCondition AlertCondition
	MemCount     int64
	MemCondition AlertCondition
}

func (s *System) updateAlerts(metrics Metrics) (alertUpdate, error) {
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
	var update alertUpdate

	var condition AlertCondition

	if metrics.CPUUsage > HighCPUThreshold {
		condition = ConditionHigh
	} else {
		condition = ConditionNormal
	}

	count, err := s.alertState.IncrementCount(
		AlertTypeHighCPU,
		condition,
	)
	if err != nil {
		return alertUpdate{}, fmt.Errorf("не удалось обновить состояние CPU-алерта: %w", err)
	}
	update.CPUCount = count
	update.CPUCondition = condition

	if metrics.MemUsage > HighMemThreshold {
		condition = ConditionHigh
	} else {
		condition = ConditionNormal
	}

	count, err = s.alertState.IncrementCount(
		AlertTypeHighMem,
		condition,
	)
	if err != nil {
		return alertUpdate{}, fmt.Errorf("не удалось обновить состояние Mem-алерта: %w", err)
	}
	update.MemCount = count
	update.MemCondition = condition

	return update, nil
}

func (s *System) processAlerts(metrics Metrics, update alertUpdate) error {
	// проверяем четыре сценария, по 2 на процессор и память (создание и резолв алерта)

	if update.CPUCondition == ConditionHigh && update.CPUCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.CPUCondition == ConditionNormal && update.CPUCount >= AlertResolveCount {
		err := s.resolveAlertIfNeeded(AlertTypeHighCPU)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	//с памятью точно также
	if update.MemCondition == ConditionHigh && update.MemCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.MemCondition == ConditionNormal && update.MemCount >= AlertResolveCount {
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
		alertsTotal.Inc()
		alertsActiveTotal.Inc()
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
		slog.Info("зарезолвлен алерт",
			"type", alert.Type,
			"value", alert.Value)
		alertsActiveTotal.Dec()
	}

	return nil
}

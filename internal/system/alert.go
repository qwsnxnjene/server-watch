package system

import (
	"fmt"
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

func (a *AlertState) Record(value float64, threshold float64) {
	if value > threshold {
		a.consecutiveHigh++
		a.consecutiveNormal = 0
	} else {
		a.consecutiveNormal++
		a.consecutiveHigh = 0
	}
}

//TODO: логировать при превышении порогов

func (a *AlertState) HighThresholdReached() bool {
	return a.consecutiveHigh >= AlertTriggerCount
}

func (a *AlertState) ResolveThresholdReached() bool {
	return a.consecutiveNormal >= AlertResolveCount
}

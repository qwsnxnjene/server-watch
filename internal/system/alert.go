package system

import "time"

type AlertType string

const (
	AlertTypeHighCPU AlertType = "HIGH_CPU"
	AlertTypeHighMem AlertType = "HIGH_MEM"
)

//TODO: GetAlerts для хэндлера

type Alert struct {
	ID         int64
	Type       AlertType
	Timestamp  time.Time
	Threshold  float64
	Resolved   bool
	ResolvedAt *time.Time
	Value      float64
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

func (a *AlertState) HighThresholdReached() bool {
	return a.consecutiveHigh >= AlertTriggerCount
}

func (a *AlertState) ResolveThresholdReached() bool {
	return a.consecutiveNormal >= AlertResolveCount
}

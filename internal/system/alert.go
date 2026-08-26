package system

import "time"

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

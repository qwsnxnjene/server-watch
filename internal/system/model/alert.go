package model

import "time"

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

type AlertCondition string

const (
	ConditionNormal AlertCondition = "normal"
	ConditionHigh   AlertCondition = "high"
)

type AlertUpdate struct {
	CPUCount     int64
	CPUCondition AlertCondition
	MemCount     int64
	MemCondition AlertCondition
}

package model

import "time"

// AlertType определяет тип системного алерта
type AlertType string

const (
	AlertTypeHighCPU AlertType = "HIGH_CPU"
	AlertTypeHighMem AlertType = "HIGH_MEM"
)

// Alert представляет сохранённый алерт и его текущее состояние
type Alert struct {
	ID         int64
	Type       AlertType
	Timestamp  time.Time
	Threshold  float64
	Resolved   bool
	ResolvedAt *time.Time
	Value      float64
}

// AlertCondition определяет условие, при котором находится мониторируемый ресурс
type AlertCondition string

const (
	ConditionNormal AlertCondition = "normal"
	ConditionHigh   AlertCondition = "high"
)

// AlertUpdate содержит текущее состояние счётчиков и условий для CPU и памяти
type AlertUpdate struct {
	CPUCount     int64
	CPUCondition AlertCondition
	MemCount     int64
	MemCondition AlertCondition
}

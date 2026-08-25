package system

import "time"

type Alert struct {
	ID         int64
	Type       string
	Timestamp  time.Time
	Threshold  float64
	Resolved   bool
	ResolvedAt *time.Time
	Value      float64
}

package system

import (
	"time"
)

type Metrics struct {
	CPUUsage   float64
	MemUsage   float64
	MemUsedMB  float64
	MemTotalMB float64
	DiskUsage  float64
	DiskUsed   float64
	DiskTotal  float64
	Timestamp  time.Time
}

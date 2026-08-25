package system

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type System struct {
	mu          sync.RWMutex
	metrics     Metrics
	lastSuccess time.Time
	lastError   error
}

func NewSystem() *System {
	return &System{}
}

// GetMetrics возвращает копию метрик
func (s *System) GetMetrics() Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.metrics
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

	s.mu.Lock()
	s.metrics = metrics
	s.lastError = nil
	s.lastSuccess = now
	s.mu.Unlock()

	log.Println("[INFO] метрики успешно обновлены")

	return nil
}

func (s *System) GetHealth() (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSuccess, s.lastError
}

package system

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readMemoryStats парсит /proc/meminfo и получает объем всей памяти, объем доступной памяти
// и вычисляет процент использования памяти.
func readMemoryStats() (float64, float64, float64, error) {
	info, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка чтения /proc/meminfo: %w", err)
	}

	lines := strings.Split(string(info), "\n")[:3]

	memTotal, err := strconv.ParseUint(strings.Fields(lines[0])[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка перевода MemTotal в int: %w", err)
	}

	memAvailable, err := strconv.ParseUint(strings.Fields(lines[2])[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка перевода MemAvailable в int: %w", err)
	}

	memUsed := memTotal - memAvailable
	memUsage := float64(memUsed) / float64(memTotal) * 100

	return float64(memTotal) / float64(1<<10), float64(memUsed) / float64(1<<10), memUsage, nil
}

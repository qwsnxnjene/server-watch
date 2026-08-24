package system

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// getDiskStats получает данные об объеме всего диска, об объеме свободной памяти на диске
// и вычисляет процент использованной памяти на диске.
func getDiskStats() (float64, float64, float64, error) {
	var stats unix.Statfs_t
	err := unix.Statfs("/", &stats)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка Statfs: %w", err)
	}

	diskAvailable := uint64(stats.Bsize) * stats.Bavail
	diskTotal := uint64(stats.Bsize) * stats.Blocks
	diskUsed := diskTotal - diskAvailable
	diskUsage := float64(diskUsed) / float64(diskTotal) * 100.0

	return float64(diskTotal) / float64(1<<30), float64(diskUsed) / float64(1<<30), diskUsage, nil
}

package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("+----------------------+----------------------+")
	fmt.Printf("| %-20s | %-20s |\n", "Metric", "Value")
	fmt.Println("+----------------------+----------------------+")
	usage, err := getCPUUsage()
	if err != nil {
		log.Fatalf("не удалось получить данные о загрузке CPU: %v", err)
	}
	fmt.Printf("| %-20s | %-20s |\n", "CPU Usage", fmt.Sprintf("%.2f%%", usage))

	totalMem, usedMem, memUsage, err := readMemoryStats()
	if err != nil {
		log.Fatalf("не удалось получить данные о памяти: %v", err)
	}
	fmt.Printf("| %-20s | %-20s ср|\n", "Memory Usage", fmt.Sprintf("%.2f%%", memUsage))
	fmt.Printf("| %-20s | %-20s |\n", "Memory", fmt.Sprintf("%.1f / %.1f MB", usedMem, totalMem))

	totalDisk, usedDisk, diskUsage, err := getDiskStats()
	if err != nil {
		log.Fatalf("не удалось получить данные о диске: %v", err)
	}
	fmt.Printf("| %-20s | %-20s |\n", "Disk Usage", fmt.Sprintf("%.2f%%", diskUsage))
	fmt.Printf("| %-20s | %-20s |\n", "Disk", fmt.Sprintf("%.1f / %.1f GB", usedDisk, totalDisk))
	fmt.Println("+----------------------+----------------------+")
}

// getDiskStats получает данные об объеме всего диска, об объеме свободной памяти на диске
// и вычисляет процент использованной памяти на диске.
func getDiskStats() (float64, float64, float64, error) {
	var stats unix.Statfs_t
	err := unix.Statfs("/", &stats)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка Statfs: %v", err)
	}

	diskAvailable := uint64(stats.Bsize) * stats.Bavail
	diskTotal := uint64(stats.Bsize) * stats.Blocks
	diskUsed := diskTotal - diskAvailable
	diskUsage := float64(diskUsed) / float64(diskTotal) * 100.0

	return float64(diskTotal) / float64(1<<30), float64(diskUsed) / float64(1<<30), diskUsage, nil
}

// readMemoryStats парсит /proc/meminfo и получает объем всей памяти, объем доступной памяти
// и вычисляет процент использования памяти.
func readMemoryStats() (float64, float64, float64, error) {
	info, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка чтения /proc/meminfo: %v", err)
	}

	lines := strings.Split(string(info), "\n")[:3]

	memTotal, err := strconv.ParseUint(strings.Fields(lines[0])[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка перевода MemTotal в int: %v", err)
	}

	memAvailable, err := strconv.ParseUint(strings.Fields(lines[2])[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ошибка перевода MemAvailable в int: %v", err)
	}

	memUsed := memTotal - memAvailable
	memUsage := float64(memUsed) / float64(memTotal) * 100

	return float64(memTotal) / float64(1<<10), float64(memUsed) / float64(1<<10), memUsage, nil
}

// getCPUUsage считывает информацию о загрузке CPU с интервалом в секунду.
// Затем по формуле высчитывает уровень загрузки CPU
func getCPUUsage() (float64, error) {
	statsFirst, err := readCPUStats()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения данных о процессоре: %v", err)
	}
	time.Sleep(time.Second)

	statsSecond, err := readCPUStats()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения данных о процессоре: %v", err)
	}

	userDelta := statsSecond[0] - statsFirst[0]
	niceDelta := statsSecond[1] - statsFirst[1]
	sysDelta := statsSecond[2] - statsFirst[2]
	idleDelta := statsSecond[3] - statsFirst[3]
	iowaitDelta := statsSecond[4] - statsFirst[4]
	irqDelta := statsSecond[5] - statsFirst[5]
	softirqDelta := statsSecond[6] - statsFirst[6]
	stealDelta := statsSecond[7] - statsFirst[7]

	cpuWorkTime := userDelta + niceDelta + sysDelta + irqDelta + softirqDelta + stealDelta
	totalTime := cpuWorkTime + idleDelta + iowaitDelta

	return float64(cpuWorkTime) / float64(totalTime) * 100.0, nil

}

// readCPUStats парсит файл /proc/stat для показателей процессора.
func readCPUStats() ([8]uint64, error) {
	info, err := os.ReadFile("/proc/stat")
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка чтения /proc/stat: %v", err)
	}

	cpu := strings.Split(string(info), "\n")[0]
	cpuFields := strings.Fields(cpu)
	if len(cpuFields) < 9 {
		return [8]uint64{}, fmt.Errorf("недостаточное количество данных о процессоре")
	}

	var stats [8]uint64

	// user - время, потраченное CPU на выполнение пользовательских процессов
	stats[0], err = strconv.ParseUint(cpuFields[1], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода user в int: %v", err)
	}

	// nice - время, потраченное CPU на выполнение процессов с пониженным приоритетом
	stats[1], err = strconv.ParseUint(cpuFields[2], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода nice в int: %v", err)
	}

	// system - время, потраченное CPU внутри ядра Linux
	stats[2], err = strconv.ParseUint(cpuFields[3], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода system в int: %v", err)
	}

	// idle - время, в которое CPU не выполнял ничего полезного для выполнения процессов
	stats[3], err = strconv.ParseUint(cpuFields[4], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода idle в int: %v", err)
	}

	//iowait - время, потраченное CPU на ожидание завершения I/O операция (диск, сеть)
	stats[4], err = strconv.ParseUint(cpuFields[5], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода iowait в int: %v", err)
	}

	//irq - время, потраченное CPU на обработку прерываний периферии
	stats[5], err = strconv.ParseUint(cpuFields[6], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода irq в int: %v", err)
	}

	//softirq - время, потраченное CPU на обработку системных прерываний
	stats[6], err = strconv.ParseUint(cpuFields[7], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода softirq в int: %v", err)
	}

	//steal - время, потраченное на ожидание выделения ресурсов от гипервизора (для виртуальных машин)
	stats[7], err = strconv.ParseUint(cpuFields[8], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода steal в int: %v", err)
	}

	return stats, nil
}

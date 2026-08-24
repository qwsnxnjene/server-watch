package system

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// getCPUUsage считывает информацию о загрузке CPU с интервалом в секунду.
// Затем по формуле высчитывает уровень загрузки CPU
func getCPUUsage() (float64, error) {
	statsFirst, err := readCPUStats()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения данных о процессоре: %w", err)
	}
	time.Sleep(time.Second)

	statsSecond, err := readCPUStats()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения данных о процессоре: %w", err)
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
		return [8]uint64{}, fmt.Errorf("ошибка чтения /proc/stat: %w", err)
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
		return [8]uint64{}, fmt.Errorf("ошибка перевода user в int: %w", err)
	}

	// nice - время, потраченное CPU на выполнение процессов с пониженным приоритетом
	stats[1], err = strconv.ParseUint(cpuFields[2], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода nice в int: %w", err)
	}

	// system - время, потраченное CPU внутри ядра Linux
	stats[2], err = strconv.ParseUint(cpuFields[3], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода system в int: %w", err)
	}

	// idle - время, в которое CPU не выполнял ничего полезного для выполнения процессов
	stats[3], err = strconv.ParseUint(cpuFields[4], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода idle в int: %w", err)
	}

	//iowait - время, потраченное CPU на ожидание завершения I/O операция (диск, сеть)
	stats[4], err = strconv.ParseUint(cpuFields[5], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода iowait в int: %w", err)
	}

	//irq - время, потраченное CPU на обработку прерываний периферии
	stats[5], err = strconv.ParseUint(cpuFields[6], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода irq в int: %w", err)
	}

	//softirq - время, потраченное CPU на обработку системных прерываний
	stats[6], err = strconv.ParseUint(cpuFields[7], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода softirq в int: %w", err)
	}

	//steal - время, потраченное на ожидание выделения ресурсов от гипервизора (для виртуальных машин)
	stats[7], err = strconv.ParseUint(cpuFields[8], 10, 64)
	if err != nil {
		return [8]uint64{}, fmt.Errorf("ошибка перевода steal в int: %w", err)
	}

	return stats, nil
}

package system

import (
	"fmt"
	"log/slog"
	"server-watch/internal/config"
	"server-watch/internal/notifications"
	"server-watch/internal/system/data"
	"server-watch/internal/system/model"
	"sync"
	"time"
)

const (
	HighCPUThreshold = 80.0
	HighMemThreshold = 90.0

	AlertTriggerCount = 3
	AlertResolveCount = 3
)

// System - системный слой, ответственный за бизнес-логику
type System struct {
	mu          sync.RWMutex
	metrics     Metrics
	lastSuccess time.Time
	lastError   error
	repository  Repository

	config     config.Config
	configPath string

	alertState        AlertStateStore
	cache             MetricsCache
	notificationQueue notifications.Queue
}

func NewSystem(
	repository Repository,
	alertState AlertStateStore,
	cfg config.Config,
	path string,
	cache MetricsCache,
	queue notifications.Queue) *System {
	return &System{
		repository:        repository,
		alertState:        alertState,
		config:            cfg,
		configPath:        path,
		cache:             cache,
		notificationQueue: queue,
	}
}

// CollectMetrics с помощью вспомогательных функций собирает свежие данные с ОС
func (s *System) CollectMetrics() error {
	usage, err := data.GetCPUUsage()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о загрузке CPU: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalMem, usedMem, memUsage, err := data.ReadMemoryStats()
	if err != nil {
		errToReturn := fmt.Errorf("не удалось получить данные о памяти: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()

		return errToReturn
	}

	totalDisk, usedDisk, diskUsage, err := data.GetDiskStats()
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

	if err := s.repository.SaveMetrics(metrics); err != nil {
		errToReturn := fmt.Errorf("не удалось сохранить метрики в БД: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	// обновляем актуальные измерения метрик на данный момент
	s.mu.Lock()
	s.metrics = metrics
	s.lastError = nil
	s.lastSuccess = metrics.Timestamp
	s.mu.Unlock()

	updatePrometheusMetrics(metrics)

	update, err := s.updateAlerts(metrics)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обновить состояния алертов: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	err = s.processAlerts(metrics, update)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обработать алерты: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	if s.cache != nil {
		err = s.cache.SetMetrics(metrics)
		if err != nil {
			slog.Warn("не удалось сохранить метрики в кэш", "error", err)
		}
	}

	slog.Info("метрики успешно обновлены")

	return nil
}

// GetAlerts возвращает список алертов с возможностью выбрать только активные с помощью флага activeOnly
func (s *System) GetAlerts(activeOnly bool) ([]model.Alert, error) {
	alerts, err := s.repository.GetAlerts(activeOnly)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список алертов: %w", err)
	}

	return alerts, nil
}

func (s *System) updateAlerts(metrics Metrics) (model.AlertUpdate, error) {
	if metrics.CPUUsage > HighCPUThreshold {
		slog.Warn(
			"превышен порог CPU",
			"value", metrics.CPUUsage,
			"threshold", HighCPUThreshold,
		)
	}

	if metrics.MemUsage > HighMemThreshold {
		slog.Warn(
			"превышен порог памяти",
			"value", metrics.MemUsage,
			"threshold", HighMemThreshold,
		)
	}

	// обновляем данные об алертах для процессора и памяти
	var update model.AlertUpdate

	var condition model.AlertCondition

	if metrics.CPUUsage > HighCPUThreshold {
		condition = model.ConditionHigh
	} else {
		condition = model.ConditionNormal
	}

	count, err := s.alertState.IncrementCount(
		model.AlertTypeHighCPU,
		condition,
	)
	if err != nil {
		return model.AlertUpdate{}, fmt.Errorf("не удалось обновить состояние CPU-алерта: %w", err)
	}
	update.CPUCount = count
	update.CPUCondition = condition

	if metrics.MemUsage > HighMemThreshold {
		condition = model.ConditionHigh
	} else {
		condition = model.ConditionNormal
	}

	count, err = s.alertState.IncrementCount(
		model.AlertTypeHighMem,
		condition,
	)
	if err != nil {
		return model.AlertUpdate{}, fmt.Errorf("не удалось обновить состояние Mem-алерта: %w", err)
	}
	update.MemCount = count
	update.MemCondition = condition

	return update, nil
}

func (s *System) processAlerts(metrics Metrics, update model.AlertUpdate) error {
	// проверяем четыре сценария, по 2 на процессор и память (создание и резолв алерта)

	if update.CPUCondition == model.ConditionHigh && update.CPUCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(model.AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.CPUCondition == model.ConditionNormal && update.CPUCount >= AlertResolveCount {
		err := s.resolveAlertIfNeeded(model.AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	//с памятью точно также
	if update.MemCondition == model.ConditionHigh && update.MemCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(model.AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.MemCondition == model.ConditionNormal && update.MemCount >= AlertResolveCount {
		err := s.resolveAlertIfNeeded(model.AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	return nil
}

func (s *System) createAlertIfNeeded(alertType model.AlertType, value float64, threshold float64) error {
	// если активного алерта на данный момент нет, то сохраняем новый, иначе не делаем ничего

	alert, err := s.repository.GetActiveAlert(alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert == nil {
		now := time.Now()

		alertToSave := model.Alert{
			Type:       alertType,
			Timestamp:  now,
			Threshold:  threshold,
			Resolved:   false,
			ResolvedAt: nil,
			Value:      value,
		}

		_, err = s.repository.SaveAlert(alertToSave)
		if err != nil {
			return fmt.Errorf("не удалось сохранить новый алерт типа %v: %w", alertType, err)
		}

		slog.Info("создан новый алерт!",
			"type", alertToSave.Type,
			"threshold", alertToSave.Threshold,
			"value", alertToSave.Value)
		alertsTotal.Inc()
		alertsActiveTotal.Inc()

		notification := notifications.Notification{
			Type:      alertType,
			Action:    notifications.ActionCreated,
			Value:     value,
			Threshold: threshold,
			Timestamp: now,
		}
		err = s.notificationQueue.Push(notification)
		if err != nil {
			slog.Error(
				"не удалось поставить уведомление в очередь",
				"error", err,
				"alert_type", alertType,
			)
		}
	}

	return nil
}

func (s *System) resolveAlertIfNeeded(alertType model.AlertType, value float64, threshold float64) error {
	// пробуем зарезолвить алерт по id

	alert, err := s.repository.GetActiveAlert(alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert != nil {
		err = s.repository.ResolveAlert(alert.ID, time.Now())
		if err != nil {
			return fmt.Errorf("не удалось зарезолвить алерт типа %v: %w", alertType, err)
		}
		slog.Info("зарезолвлен алерт",
			"type", alert.Type,
			"value", alert.Value)
		alertsActiveTotal.Dec()

		notification := notifications.Notification{
			Type:      alertType,
			Action:    notifications.ActionResolved,
			Value:     value,
			Threshold: threshold,
			Timestamp: time.Now(),
		}
		err = s.notificationQueue.Push(notification)
		if err != nil {
			slog.Error(
				"не удалось поставить уведомление в очередь",
				"error", err,
				"alert_type", alertType,
			)
		}
	}

	return nil
}

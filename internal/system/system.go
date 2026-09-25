package system

import (
	"context"
	"fmt"
	"log/slog"
	"server-watch/internal/config"
	"server-watch/internal/notifications"
	"server-watch/internal/prometheus"
	"server-watch/internal/system/data"
	"server-watch/internal/system/model"
	"sync"
	"time"
)

// Пороговые значения и количество последовательных измерений,
// необходимые для создания и разрешения алерта
const (
	HighCPUThreshold = 80.0
	HighMemThreshold = 90.0

	AlertTriggerCount = 3
	AlertResolveCount = 3
)

// System реализует бизнес-логику мониторинга системных метрик и алертов
type System struct {
	mu          sync.RWMutex
	metrics     model.Metrics
	lastSuccess time.Time
	lastError   error
	repository  Repository

	config     config.Config
	configPath string

	alertState        AlertStateStore
	cache             MetricsCache
	notificationQueue notifications.Queue
}

// NewSystem создаёт системный слой с указанными хранилищами,
// конфигурацией и очередью уведомлений
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

// CollectMetrics собирает системные метрики, сохраняет их,
// обновляет состояние алертов и передаёт уведомления в очередь
func (s *System) CollectMetrics(ctx context.Context) error {
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

	metrics := model.Metrics{
		CPUUsage:   usage,
		MemUsage:   memUsage,
		MemUsedMB:  usedMem,
		MemTotalMB: totalMem,
		DiskUsage:  diskUsage,
		DiskUsed:   usedDisk,
		DiskTotal:  totalDisk,
		Timestamp:  now,
	}

	if err := s.repository.SaveMetrics(ctx, metrics); err != nil {
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

	prometheus.UpdatePrometheusMetrics(metrics)

	update, err := s.updateAlerts(ctx, metrics)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обновить состояния алертов: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	err = s.processAlerts(ctx, metrics, update)
	if err != nil {
		errToReturn := fmt.Errorf("не удалось обработать алерты: %w", err)
		s.mu.Lock()
		s.lastError = errToReturn
		s.mu.Unlock()
		return errToReturn
	}

	if s.cache != nil {
		err = s.cache.SetMetrics(ctx, metrics)
		if err != nil {
			slog.Warn("не удалось сохранить метрики в кэш", "error", err)
		}
	}

	slog.Info("метрики успешно обновлены")

	return nil
}

// GetAlerts возвращает список алертов.
// При activeOnly == true возвращаются только активные алерты
func (s *System) GetAlerts(ctx context.Context, activeOnly bool) ([]model.Alert, error) {
	alerts, err := s.repository.GetAlerts(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список алертов: %w", err)
	}

	return alerts, nil
}

// updateAlerts обновляет последовательные состояния алертов для CPU и памяти
func (s *System) updateAlerts(ctx context.Context, metrics model.Metrics) (model.AlertUpdate, error) {
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

	var update model.AlertUpdate

	var condition model.AlertCondition

	if metrics.CPUUsage > HighCPUThreshold {
		condition = model.ConditionHigh
	} else {
		condition = model.ConditionNormal
	}

	count, err := s.alertState.IncrementCount(
		ctx,
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
		ctx,
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

// processAlerts создаёт или разрешает алерты после достижения
// необходимого количества последовательных измерений
func (s *System) processAlerts(ctx context.Context, metrics model.Metrics, update model.AlertUpdate) error {
	if update.CPUCondition == model.ConditionHigh && update.CPUCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(ctx, model.AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.CPUCondition == model.ConditionNormal && update.CPUCount >= AlertResolveCount {
		err := s.resolveAlertIfNeeded(ctx, model.AlertTypeHighCPU, metrics.CPUUsage, HighCPUThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	//с памятью точно также
	if update.MemCondition == model.ConditionHigh && update.MemCount >= AlertTriggerCount {
		err := s.createAlertIfNeeded(ctx, model.AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	if update.MemCondition == model.ConditionNormal && update.MemCount >= AlertResolveCount {
		err := s.resolveAlertIfNeeded(ctx, model.AlertTypeHighMem, metrics.MemUsage, HighMemThreshold)
		if err != nil {
			return fmt.Errorf("не удалось обработать алерт: %w", err)
		}
	}

	return nil
}

// createAlertIfNeeded создаёт алерт, если для указанного типа
// ещё нет активного алерта
func (s *System) createAlertIfNeeded(ctx context.Context, alertType model.AlertType, value float64, threshold float64) error {
	alert, err := s.repository.GetActiveAlert(ctx, alertType)
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

		_, err = s.repository.SaveAlert(ctx, alertToSave)
		if err != nil {
			return fmt.Errorf("не удалось сохранить новый алерт типа %v: %w", alertType, err)
		}

		slog.Info("создан новый алерт!",
			"type", alertToSave.Type,
			"threshold", alertToSave.Threshold,
			"value", alertToSave.Value)
		prometheus.AlertsTotal.Inc()
		prometheus.AlertsActiveTotal.Inc()

		notification := notifications.Notification{
			Type:      alertType,
			Action:    notifications.ActionCreated,
			Value:     value,
			Threshold: threshold,
			Timestamp: now,
		}
		err = s.notificationQueue.Push(ctx, notification)
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

// resolveAlertIfNeeded разрешает активный алерт указанного типа,
// если он существует
func (s *System) resolveAlertIfNeeded(ctx context.Context, alertType model.AlertType, value float64, threshold float64) error {
	alert, err := s.repository.GetActiveAlert(ctx, alertType)
	if err != nil {
		return fmt.Errorf("не удалось получить активный алерт типа %v: %w", alertType, err)
	}
	if alert != nil {
		err = s.repository.ResolveAlert(ctx, alert.ID, time.Now())
		if err != nil {
			return fmt.Errorf("не удалось зарезолвить алерт типа %v: %w", alertType, err)
		}
		slog.Info("зарезолвлен алерт",
			"type", alert.Type,
			"value", alert.Value)
		prometheus.AlertsActiveTotal.Dec()

		notification := notifications.Notification{
			Type:      alertType,
			Action:    notifications.ActionResolved,
			Value:     value,
			Threshold: threshold,
			Timestamp: time.Now(),
		}
		err = s.notificationQueue.Push(ctx, notification)
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

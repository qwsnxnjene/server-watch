package system

import (
	"context"
	"fmt"
	"log/slog"
	"server-watch/internal/system/model"
)

// FallbackAlertStateStore использует Redis как основное хранилище состояния алертов
// и переключается на in-memory хранилище при его недоступности
type FallbackAlertStateStore struct {
	memory AlertStateBackend
	redis  AlertStateBackend

	redisFailed bool
}

// NewFallbackAlertStateStore создаёт хранилище с Redis в качестве основного
// и in-memory хранилищем для fallback
func NewFallbackAlertStateStore(mem, redis AlertStateBackend) *FallbackAlertStateStore {
	return &FallbackAlertStateStore{
		memory: mem,
		redis:  redis,
	}
}

// IncrementCount увеличивает счётчик состояния через Redis,
// а при его недоступности использует in-memory хранилище.
// После восстановления Redis данные синхронизируются перед продолжением работы
func (f *FallbackAlertStateStore) IncrementCount(ctx context.Context, alertType model.AlertType, condition model.AlertCondition) (int64, error) {
	var redisErr error

	// Если Redis ранее падал - сначала пытаемся восстановить
	if f.redisFailed {
		if err := f.resyncAll(ctx); err == nil {
			f.redisFailed = false
			slog.Info("соединение с Redis восстановлено, данные синхронизированы")
		} else {
			slog.Info("не удалось синхронизировать данные с Redis, продолжаем работу с in-memory",
				"error", err)
			redisErr = err
		}
	}

	// Если Redis восстановлен или раньше не падал
	// пробуем выполнить обычную операцию
	if !f.redisFailed {
		redisValue, err := f.redis.IncrementCount(ctx, alertType, condition)
		if err == nil {
			return redisValue, nil
		}

		redisErr = err
		f.redisFailed = true
	}

	// Redis недоступен, тогда fallback на in-memory
	memoryValue, memoryErr := f.memory.IncrementCount(ctx, alertType, condition)
	if memoryErr != nil {
		return 0, fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memoryErr,
		)
	}

	return memoryValue, nil
}

// IsActive возвращает статус алерта, используя Redis или in-memory fallback
// при недоступности Redis
func (f *FallbackAlertStateStore) IsActive(ctx context.Context, alertType model.AlertType) (bool, error) {
	redisActive, redisErr := f.redis.IsActive(ctx, alertType)
	if redisErr == nil {
		return redisActive, nil
	}

	memActive, memErr := f.memory.IsActive(ctx, alertType)
	if memErr != nil {
		return false, fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memErr,
		)
	}

	return memActive, nil
}

// SetActive устанавливает статус алерта в Redis или in-memory fallback,
// если Redis недоступен
func (f *FallbackAlertStateStore) SetActive(ctx context.Context, alertType model.AlertType, active bool) error {
	if f.redisFailed {
		if err := f.resyncAll(ctx); err == nil {
			f.redisFailed = false
			slog.Info("соединение с Redis восстановлено, данные синхронизированы")
		} else {
			slog.Info("не удалось синхронизировать данные с Redis, продолжаем работу с in-memory",
				"error", err)

			memErr := f.memory.SetActive(ctx, alertType, active)
			if memErr != nil {
				return fmt.Errorf(
					"ошибка синхронизации redis: %v; ошибка in-memory: %w",
					err,
					memErr,
				)
			}

			return nil
		}
	}

	redisErr := f.redis.SetActive(ctx, alertType, active)
	if redisErr == nil {
		return nil
	}

	f.redisFailed = true

	memErr := f.memory.SetActive(ctx, alertType, active)
	if memErr != nil {
		return fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memErr,
		)
	}

	return nil
}

// resyncAll переносит состояния всех поддерживаемых типов алертов
// из in-memory хранилища обратно в Redis
func (f *FallbackAlertStateStore) resyncAll(ctx context.Context) error {
	for _, alertType := range []model.AlertType{model.AlertTypeHighMem, model.AlertTypeHighCPU} {

		state, err := f.memory.GetState(ctx, alertType)
		if err != nil {
			return fmt.Errorf("не удалось получить состояние из in-memory: %w", err)
		}

		err = f.redis.SetState(ctx, alertType, state)
		if err != nil {
			return fmt.Errorf("не удалось установить состояние в redis: %w", err)
		}
	}
	return nil
}

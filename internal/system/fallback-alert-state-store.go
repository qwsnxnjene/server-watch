package system

import (
	"fmt"
	"log/slog"
)

type FallbackAlertStateStore struct {
	memory AlertStateBackend
	redis  AlertStateBackend

	redisFailed bool
}

func NewFallbackAlertStateStore(mem, redis AlertStateBackend) *FallbackAlertStateStore {
	return &FallbackAlertStateStore{
		memory: mem,
		redis:  redis,
	}
}

func (f *FallbackAlertStateStore) IncrementCount(alertType AlertType, condition AlertCondition) (int64, error) {
	var redisErr error

	// Если Redis ранее падал - сначала пытаемся восстановить
	if f.redisFailed {
		if err := f.resyncAll(); err == nil {
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
		redisValue, err := f.redis.IncrementCount(alertType, condition)
		if err == nil {
			return redisValue, nil
		}

		redisErr = err
		f.redisFailed = true
	}

	// Redis недоступен, тогда fallback на in-memory
	memoryValue, memoryErr := f.memory.IncrementCount(alertType, condition)
	if memoryErr != nil {
		return 0, fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memoryErr,
		)
	}

	return memoryValue, nil
}

func (f *FallbackAlertStateStore) IsActive(alertType AlertType) (bool, error) {
	redisActive, redisErr := f.redis.IsActive(alertType)
	if redisErr == nil {
		return redisActive, nil
	}

	memActive, memErr := f.memory.IsActive(alertType)
	if memErr != nil {
		return false, fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memErr,
		)
	}

	return memActive, nil
}

func (f *FallbackAlertStateStore) SetActive(alertType AlertType, active bool) error {
	if f.redisFailed {
		if err := f.resyncAll(); err == nil {
			f.redisFailed = false
			slog.Info("соединение с Redis восстановлено, данные синхронизированы")
		} else {
			slog.Info("не удалось синхронизировать данные с Redis, продолжаем работу с in-memory",
				"error", err)

			memErr := f.memory.SetActive(alertType, active)
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

	redisErr := f.redis.SetActive(alertType, active)
	if redisErr == nil {
		return nil
	}

	f.redisFailed = true

	memErr := f.memory.SetActive(alertType, active)
	if memErr != nil {
		return fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memErr,
		)
	}

	return nil
}

func (f *FallbackAlertStateStore) resyncAll() error {
	for _, alertType := range []AlertType{AlertTypeHighMem, AlertTypeHighCPU} {

		state, err := f.memory.GetState(alertType)
		if err != nil {
			return fmt.Errorf("не удалось получить состояние из in-memory: %w", err)
		}

		err = f.redis.SetState(alertType, state)
		if err != nil {
			return fmt.Errorf("не удалось установить состояние в redis: %w", err)
		}
	}
	return nil
}

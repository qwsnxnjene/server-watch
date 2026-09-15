package system

import (
	"fmt"
)

type FallbackAlertStateStore struct {
	memory AlertStateStore
	redis  AlertStateStore
}

func NewFallbackAlertStateStore(mem, redis AlertStateStore) *FallbackAlertStateStore {
	return &FallbackAlertStateStore{
		memory: mem,
		redis:  redis,
	}
}

func (f *FallbackAlertStateStore) IncrementCount(alertType AlertType) (int64, error) {
	redisValue, redisErr := f.redis.IncrementCount(alertType)

	if redisErr == nil {
		return redisValue, nil
	}

	memoryValue, memoryErr := f.memory.IncrementCount(alertType)
	if memoryErr != nil {
		return 0, fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memoryErr,
		)
	}

	return memoryValue, nil
}

func (f *FallbackAlertStateStore) ResetCount(alertType AlertType) error {
	redisErr := f.redis.ResetCount(alertType)
	if redisErr == nil {
		return nil
	}

	memErr := f.memory.ResetCount(alertType)
	if memErr != nil {
		return fmt.Errorf(
			"ошибка redis: %v; ошибка in-memory: %w",
			redisErr,
			memErr,
		)
	}

	return nil
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
	redisErr := f.redis.SetActive(alertType, active)
	if redisErr == nil {
		return nil
	}

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

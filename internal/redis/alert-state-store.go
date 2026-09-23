package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"server-watch/internal/system/model"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed set_state.lua
var setStateScript string

// RedisAlertStateStore хранит состояния алертов в Redis
type RedisAlertStateStore struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisAlertStateStore создаёт хранилище состояний алертов в Redis
func NewRedisAlertStateStore(client *redis.Client, ttl time.Duration, prefix string) *RedisAlertStateStore {
	return &RedisAlertStateStore{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

func (r *RedisAlertStateStore) alertKey(alertType model.AlertType, suffix string) (string, error) {
	switch alertType {
	case model.AlertTypeHighCPU:
		return r.prefix + "cpu:" + suffix, nil
	case model.AlertTypeHighMem:
		return r.prefix + "mem:" + suffix, nil
	default:
		return "", errors.New("некорректный тип алерта")
	}
}

// IncrementCount увеличивает счётчик при сохранении условия
// или сбрасывает его при изменении условия
func (r *RedisAlertStateStore) IncrementCount(ctx context.Context, alertType model.AlertType, condition model.AlertCondition) (int64, error) {
	keyCond, err := r.alertKey(alertType, "condition")
	if err != nil {
		return 0, err
	}

	keyCount, err := r.alertKey(alertType, "count")
	if err != nil {
		return 0, err
	}

	cond, err := r.client.Get(ctx, keyCond).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("не удалось получить состояние: %w", err)
	}

	if cond == string(condition) {
		// состояния совпадают - просто увеличиваем счетчик алерта
		res, err := r.client.Incr(ctx, keyCount).Result()
		if err != nil {
			return 0, fmt.Errorf("не удалось инкрементировать счетчик алерта: %w", err)
		}
		err = r.client.Expire(ctx, keyCount, r.ttl).Err()
		if err != nil {
			return 0, fmt.Errorf("не удалось установить TTL у счетчика алерта: %w", err)
		}
		err = r.client.Expire(ctx, keyCond, r.ttl).Err()
		if err != nil {
			return 0, fmt.Errorf("не удалось установить TTL у состояния счетчика: %w", err)
		}

		return res, nil
	} else {
		// состояния разные - устанавливаем новое состояние и обновляем счетчик алерта
		err = r.client.Set(ctx, keyCond, string(condition), r.ttl).Err()
		if err != nil {
			return 0, fmt.Errorf("не удалось установить новое состояние: %w", err)
		}
		_, err := r.client.Set(ctx, keyCount, 1, r.ttl).Result()
		if err != nil {
			return 0, fmt.Errorf("не удалось установить значение счетчика алерта: %w", err)
		}

		return 1, nil
	}
}

// IsActive проверяет, активен ли алерт заданного типа
func (r *RedisAlertStateStore) IsActive(ctx context.Context, alertType model.AlertType) (bool, error) {
	key, err := r.alertKey(alertType, "active")
	if err != nil {
		return false, err
	}

	res, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("не удалось получить информацию об активности алерта: %w", err)
	}

	if res == "1" {
		return true, nil
	}

	return false, nil
}

// SetActive устанавливает статус алерта заданного типа
func (r *RedisAlertStateStore) SetActive(ctx context.Context, alertType model.AlertType, active bool) error {
	key, err := r.alertKey(alertType, "active")
	if err != nil {
		return err
	}

	var value int
	if active {
		value = 1
	}

	err = r.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("не удалось поменять статус алерта: %w", err)
	}

	return nil
}

// SetState атомарно сохраняет полное состояние алерта в Redis
func (r *RedisAlertStateStore) SetState(ctx context.Context, alertType model.AlertType, state model.AlertState) error {
	keyCond, err := r.alertKey(alertType, "condition")
	if err != nil {
		return err
	}

	keyCount, err := r.alertKey(alertType, "count")
	if err != nil {
		return err
	}

	keyActive, err := r.alertKey(alertType, "active")
	if err != nil {
		return err
	}

	var value int
	if state.Active {
		value = 1
	}

	err = r.client.Eval(
		ctx,
		setStateScript,
		[]string{keyCond, keyCount, keyActive},
		string(state.Condition),
		state.Count,
		value,
		int64(r.ttl.Seconds()),
	).Err()
	if err != nil {
		return fmt.Errorf("не удалось установить состояние алерта: %w", err)
	}

	return nil
}

func (r *RedisAlertStateStore) GetState(ctx context.Context, alertType model.AlertType) (model.AlertState, error) {
	return model.AlertState{}, nil
}

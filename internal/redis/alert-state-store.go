package redis

import (
	"context"
	"errors"
	"fmt"
	"server-watch/internal/system"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisAlertStateStore struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewRedisAlertStateStore(client *redis.Client, ttl time.Duration, prefix string) *RedisAlertStateStore {
	return &RedisAlertStateStore{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

func (r *RedisAlertStateStore) alertKey(alertType system.AlertType, suffix string) (string, error) {
	switch alertType {
	case system.AlertTypeHighCPU:
		return r.prefix + "cpu:" + suffix, nil
	case system.AlertTypeHighMem:
		return r.prefix + "mem:" + suffix, nil
	default:
		return "", errors.New("некорректный тип алерта")
	}
}

func (r *RedisAlertStateStore) IncrementCount(alertType system.AlertType) (int64, error) {
	key, err := r.alertKey(alertType, "count")
	if err != nil {
		return 0, err
	}

	res, err := r.client.Incr(context.Background(), key).Result()
	if err != nil {
		return 0, fmt.Errorf("не удалось инкрементировать счетчик алерта: %w", err)
	}
	err = r.client.Expire(context.Background(), key, r.ttl).Err()
	if err != nil {
		return 0, fmt.Errorf("не удалось установить TTL у счетчика алерта: %w", err)
	}

	return res, nil
}

func (r *RedisAlertStateStore) ResetCount(alertType system.AlertType) error {
	key, err := r.alertKey(alertType, "count")
	if err != nil {
		return err
	}

	if err := r.client.Set(context.Background(), key, 0, 0).Err(); err != nil {
		return fmt.Errorf("не удалось сбросить счетчик алерта: %w", err)
	}

	return nil
}

func (r *RedisAlertStateStore) IsActive(alertType system.AlertType) (bool, error) {
	key, err := r.alertKey(alertType, "active")
	if err != nil {
		return false, err
	}

	res, err := r.client.Get(context.Background(), key).Result()
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

func (r *RedisAlertStateStore) SetActive(alertType system.AlertType, active bool) error {
	key, err := r.alertKey(alertType, "active")
	if err != nil {
		return err
	}

	var value int
	if active {
		value = 1
	}

	err = r.client.Set(context.Background(), key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("не удалось поменять статус алерта: %w", err)
	}

	return nil
}

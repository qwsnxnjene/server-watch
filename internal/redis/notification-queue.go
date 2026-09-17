package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"server-watch/internal/notifications"

	"github.com/redis/go-redis/v9"
)

type RedisNotificationQueue struct {
	client *redis.Client
	prefix string
}

func NewRedisNotificationQueue(client *redis.Client, prefix string) *RedisNotificationQueue {
	return &RedisNotificationQueue{
		client: client,
		prefix: prefix,
	}
}

func (r *RedisNotificationQueue) queueKey() string {
	return r.prefix + "notifications:queue"
}

func (r *RedisNotificationQueue) Push(notification notifications.Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать уведомление: %w", err)
	}

	if err := r.client.RPush(context.Background(), r.queueKey(), string(data)).Err(); err != nil {
		return fmt.Errorf("не удалось добавить уведомление в очередь Redis: %w", err)
	}

	return nil
}

func (r *RedisNotificationQueue) Consume(ctx context.Context) (notifications.Notification, error) {
	data, err := r.client.BRPop(ctx, 0, r.queueKey()).Result()
	if err != nil {
		return notifications.Notification{},
			fmt.Errorf("не удалось получить уведомление из Redis: %w", err)
	}

	if len(data) < 2 {
		return notifications.Notification{}, errors.New("некорректная длина списка уведомлений")
	}

	var notification notifications.Notification
	err = json.Unmarshal([]byte(data[1]), &notification)
	if err != nil {
		return notifications.Notification{}, fmt.Errorf("ошибка декодирования уведомления: %w", err)
	}

	return notification, nil
}

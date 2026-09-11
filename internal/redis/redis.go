package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func StartRedisHealthCheck(ctx context.Context, client *redis.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := client.Ping(ctx).Err(); err != nil {
				slog.Warn("redis недоступен", "error", err)
				continue
			}

			slog.Info("redis доступен")

		case <-ctx.Done():
			slog.Info("проверка redis остановлена")
			return
		}
	}
}

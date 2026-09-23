package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/logging"
)

// NewClient создаёт Redis-клиент с настройками подключения по умолчанию
func NewClient() *redis.Client {
	logging.Disable()

	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

// StartRedisHealthCheck периодически проверяет доступность Redis
// и завершает работу при отмене контекста
func StartRedisHealthCheck(ctx context.Context, client *redis.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)

			err := client.Ping(pingCtx).Err()

			cancel()

			if err != nil {
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

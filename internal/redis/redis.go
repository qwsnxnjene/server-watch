package redis

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/logging"
)

// NewClient создаёт Redis-клиент с настройками подключения по умолчанию
func NewClient() *redis.Client {
	logging.Disable()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return redis.NewClient(&redis.Options{
		Addr:     redisAddr,
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

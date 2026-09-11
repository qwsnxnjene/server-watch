package redis

import (
	"context"
	"errors"
	"server-watch/internal/system"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("не удалось подключиться к Redis: %v", err)
	}

	keys := []string{
		"test:metrics:cpu",
		"test:metrics:mem_used",
		"test:metrics:mem_total",
		"test:metrics:disk_used",
		"test:metrics:disk_total",
		"test:metrics:timestamp",
	}

	cleanup := func() {
		ctx := context.Background()

		if err := client.Del(ctx, keys...).Err(); err != nil {
			t.Logf("не удалось очистить тестовые ключи: %v", err)
		}
	}

	cleanup()

	t.Cleanup(cleanup)

	return client
}

func TestNewClient(t *testing.T) {
	redisClient, err := NewClient()
	if err != nil {
		t.Fatalf("не удалось подключиться к Redis: %v", err)
	}

	if redisClient == nil {
		t.Fatalf("получен пустой клиент Redis")
	}
}

func TestRedisMetricsCache_SetGetMetrics(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	now := time.Now().UTC().Truncate(time.Second)

	metrics := system.Metrics{
		CPUUsage:   30.3,
		MemUsage:   50.0,
		MemUsedMB:  1000,
		MemTotalMB: 2000,
		DiskUsage:  80.0,
		DiskUsed:   80,
		DiskTotal:  100,
		Timestamp:  now,
	}

	err := metricsCache.SetMetrics(metrics)
	if err != nil {
		t.Fatalf("ошибка SetMetrics: %v", err)
	}

	metricsGot, err := metricsCache.GetMetrics()
	if err != nil {
		t.Fatalf("ошибка GetMetrics: %v", err)
	}

	if metricsGot != metrics {
		t.Fatalf("ожидали %+v, получили %+v", metrics, metricsGot)
	}
}

func TestRedisMetricsCache_SetMetrics(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	now := time.Now().UTC().Truncate(time.Second)

	metrics := system.Metrics{
		CPUUsage:   30.3,
		MemUsage:   50.0,
		MemUsedMB:  1000,
		MemTotalMB: 2000,
		DiskUsage:  80.0,
		DiskUsed:   80,
		DiskTotal:  100,
		Timestamp:  now,
	}

	err := metricsCache.SetMetrics(metrics)
	if err != nil {
		t.Fatalf("ошибка SetMetrics: %v", err)
	}

	tests := []struct {
		key  string
		want float64
	}{
		{"test:metrics:cpu", metrics.CPUUsage},
		{"test:metrics:mem_used", metrics.MemUsedMB},
		{"test:metrics:mem_total", metrics.MemTotalMB},
		{"test:metrics:disk_used", metrics.DiskUsed},
		{"test:metrics:disk_total", metrics.DiskTotal},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, err := metricsCache.client.Get(context.Background(), tt.key).Result()
			if err != nil {
				t.Fatalf("не удалось получить %v: %v", tt.key, err)
			}
			got, err := strconv.ParseFloat(value, 64)
			if err != nil {
				t.Fatalf("не удалось спарсить %v: %v", tt.key, err)
			}

			if got != tt.want {
				t.Fatalf("ожидали %v = %v, получили %v", tt.key, tt.want, got)
			}
		})
	}

}

func TestRedisMetricsCache_SetMetrics_TimeStamp(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	now := time.Now().UTC().Truncate(time.Second)

	metrics := system.Metrics{
		Timestamp: now,
	}

	err := metricsCache.SetMetrics(metrics)
	if err != nil {
		t.Fatalf("ошибка SetMetrics: %v", err)
	}

	value, err := metricsCache.client.Get(context.Background(), "test:metrics:timestamp").Result()
	if err != nil {
		t.Fatalf("не удалось получить metrics:timestamp: %v", err)
	}

	got, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("не удалось спарсить metrics:timestamp: %v", err)
	}

	if got != now {
		t.Fatalf("ожидали metrics:timestamp = %v, получили %v", now, got)
	}
}

func TestRedisMetricsCache_SetMetrics_TTL(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	metrics := system.Metrics{
		CPUUsage: 30.3,
	}

	err := metricsCache.SetMetrics(metrics)
	if err != nil {
		t.Fatalf("ошибка SetMetrics: %v", err)
	}

	ttl, err := metricsCache.client.TTL(
		context.Background(),
		"test:metrics:cpu",
	).Result()
	if err != nil {
		t.Fatalf("не удалось получить TTL: %v", err)
	}

	if ttl <= 0 || ttl > metricsCache.ttl {
		t.Fatalf("ожидали TTL от 0 до %v, получили %v", metricsCache.ttl, ttl)
	}
}

func TestRedisMetricsCache_GetMetrics_CacheMiss(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	_, err := metricsCache.GetMetrics()
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("ожидали ошибку %v, получили %v", ErrCacheMiss, err)
	}
}

func TestRedisMetricsCache_GetMetrics_InvalidValue(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	err := client.Set(context.Background(), "test:metrics:cpu", "abc", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}

	_, err = metricsCache.GetMetrics()
	if err == nil {
		t.Fatal("ожидали ошибку, получили nil")
	}

	if errors.Is(err, ErrCacheMiss) {
		t.Fatalf("ожидали ошибку парсинга, получили ErrCacheMiss: %v", err)
	}
}

func TestRedisMetricsCache_GetMetrics_InvalidTimestamp(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	err := client.Set(context.Background(), "test:metrics:cpu", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_total", "80.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_total", "100.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:timestamp", "invalid", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}

	_, err = metricsCache.GetMetrics()
	if err == nil {
		t.Fatal("ожидали ошибку, получили nil")
	}

	if errors.Is(err, ErrCacheMiss) {
		t.Fatalf("ожидали ошибку парсинга, получили ErrCacheMiss: %v", err)
	}
}

func TestRedisMetricsCache_GetMetrics_ZeroMemTotal(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	err := client.Set(context.Background(), "test:metrics:cpu", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_total", "0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_total", "100.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:timestamp", time.Now().UTC().Format(time.RFC3339), time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}

	_, err = metricsCache.GetMetrics()
	if err == nil {
		t.Fatal("ожидали ошибку, получили nil")
	}

	if errors.Is(err, ErrCacheMiss) {
		t.Fatalf("ожидали ошибку деления на ноль, получили ErrCacheMiss: %v", err)
	}
}

func TestRedisMetricsCache_GetMetrics_ZeroDiskTotal(t *testing.T) {
	client := newTestRedis(t)

	metricsCache := NewRedisMetricsCache(client, 30*time.Second, "test:metrics:")

	err := client.Set(context.Background(), "test:metrics:cpu", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:mem_total", "40.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_used", "30.0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:disk_total", "0", time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}
	err = client.Set(context.Background(), "test:metrics:timestamp", time.Now().UTC().Format(time.RFC3339), time.Minute).Err()
	if err != nil {
		t.Fatalf("не удалось установить тестовое значение: %v", err)
	}

	_, err = metricsCache.GetMetrics()
	if err == nil {
		t.Fatal("ожидали ошибку, получили nil")
	}

	if errors.Is(err, ErrCacheMiss) {
		t.Fatalf("ожидали ошибку деления на ноль, получили ErrCacheMiss: %v", err)
	}
}

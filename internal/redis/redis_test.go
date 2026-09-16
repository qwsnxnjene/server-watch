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

	client := NewClient()

	keys := []string{
		"test:metrics:cpu",
		"test:metrics:mem_used",
		"test:metrics:mem_total",
		"test:metrics:disk_used",
		"test:metrics:disk_total",
		"test:metrics:timestamp",
		"test:alert:cpu:count",
		"test:alert:cpu:active",
		"test:alert:mem:count",
		"test:alert:mem:active",
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
	client := NewClient()

	if client == nil {
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
	if !errors.Is(err, system.ErrCacheMiss) {
		t.Fatalf("ожидали ошибку %v, получили %v", system.ErrCacheMiss, err)
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

	if errors.Is(err, system.ErrCacheMiss) {
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

	if errors.Is(err, system.ErrCacheMiss) {
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

	if errors.Is(err, system.ErrCacheMiss) {
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

	if errors.Is(err, system.ErrCacheMiss) {
		t.Fatalf("ожидали ошибку деления на ноль, получили ErrCacheMiss: %v", err)
	}
}

func TestRedisAlertStateStore_IncrementCount(t *testing.T) {
	client := newTestRedis(t)

	stateStore := NewRedisAlertStateStore(client, time.Minute, "test:alert:")

	res, err := stateStore.IncrementCount(system.AlertTypeHighCPU, system.ConditionHigh)
	if err != nil {
		t.Fatalf("ошибка инкрементирования счетчика алерта: %v", err)
	}
	if res != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", res)
	}

	res, err = stateStore.IncrementCount(system.AlertTypeHighCPU, system.ConditionHigh)
	if err != nil {
		t.Fatalf("ошибка инкрементирования счетчика алерта: %v", err)
	}
	if res != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", res)
	}

	res, err = stateStore.IncrementCount(system.AlertTypeHighCPU, system.ConditionNormal)
	if err != nil {
		t.Fatalf("ошибка инкрементирования счетчика алерта: %v", err)
	}
	if res != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", res)
	}

	res, err = stateStore.IncrementCount(system.AlertTypeHighCPU, system.ConditionNormal)
	if err != nil {
		t.Fatalf("ошибка инкрементирования счетчика алерта: %v", err)
	}
	if res != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", res)
	}

	res, err = stateStore.IncrementCount(system.AlertTypeHighCPU, system.ConditionHigh)
	if err != nil {
		t.Fatalf("ошибка инкрементирования счетчика алерта: %v", err)
	}
	if res != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", res)
	}

	condition, err := client.Get(
		context.Background(),
		"test:alert:cpu:condition",
	).Result()
	if err != nil {
		t.Fatalf("не удалось получить состояние алерта: %v", err)
	}

	if condition != string(system.ConditionHigh) {
		t.Fatalf("ожидали состояние = %q, получили %q",
			system.ConditionHigh,
			condition,
		)
	}

	ttl, err := client.TTL(context.Background(), "test:alert:cpu:count").Result()
	if err != nil {
		t.Fatalf("не удалось получить информацию о TTL: %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Fatalf("ожидали TTL в промежутке от 0 до 1 минуты, получили %v", ttl)
	}

	ttl, err = client.TTL(context.Background(), "test:alert:cpu:condition").Result()
	if err != nil {
		t.Fatalf("не удалось получить информацию о TTL: %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Fatalf("ожидали TTL в промежутке от 0 до 1 минуты, получили %v", ttl)
	}
}

func TestRedisAlertStateStore_SetActive(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "true",
			want: true,
		},
		{
			name: "false",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestRedis(t)
			stateStore := NewRedisAlertStateStore(client, time.Minute, "test:alert:")

			err := stateStore.SetActive(system.AlertTypeHighCPU, tt.want)
			if err != nil {
				t.Fatalf("не удалось изменить статус алерта: %v", err)
			}

			got, err := stateStore.IsActive(system.AlertTypeHighCPU)
			if err != nil {
				t.Fatalf("не удалось прочитать статус алерта: %v", err)
			}

			if got != tt.want {
				t.Fatalf("ожидался статус алерта = %v, получили %v", tt.want, got)
			}

			ttl, err := client.TTL(context.Background(), "test:alert:cpu:active").Result()
			if err != nil {
				t.Fatalf("не удалось получить информацию о TTL алерта: %v", err)
			}

			if ttl != -1 {
				t.Fatalf("ожидали отсутствие TTL (-1), получили %v", ttl)
			}
		})
	}
}

func TestRedisAlertStateStore_IsActive_Missing(t *testing.T) {
	client := newTestRedis(t)
	stateStore := NewRedisAlertStateStore(client, time.Minute, "test:alert:")

	active, err := stateStore.IsActive(system.AlertTypeHighCPU)
	if err != nil {
		t.Fatalf("ошибка получения статуса алерта: %v", err)
	}
	if active {
		t.Fatalf("ожидали false, получили true")
	}
}

func TestRedisAlertStateStore_InvalidAlertType(t *testing.T) {
	client := newTestRedis(t)
	stateStore := NewRedisAlertStateStore(client, time.Minute, "test:alert:")

	_, err := stateStore.IncrementCount(system.AlertType("UNKNOWN"), system.ConditionHigh)
	if err == nil {
		t.Fatal("ожидали ошибку для неверного типа алерта")
	}
}

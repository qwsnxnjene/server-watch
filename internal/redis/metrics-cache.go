package redis

import (
	"context"
	"errors"
	"fmt"
	"server-watch/internal/system"
	"server-watch/internal/system/model"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisMetricsCache хранит последние системные метрики в Redis с ограниченным временем жизни
type RedisMetricsCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisMetricsCache создаёт Redis-кэш для системных метрик
func NewRedisMetricsCache(client *redis.Client, ttl time.Duration, prefix string) *RedisMetricsCache {
	return &RedisMetricsCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

// SetMetrics сохраняет текущие системные метрики в Redis
func (r *RedisMetricsCache) SetMetrics(ctx context.Context, metrics model.Metrics) error {
	err := r.setMetric(ctx, r.prefix+"cpu", fmt.Sprintf("%.2f", metrics.CPUUsage))
	if err != nil {
		return err
	}

	err = r.setMetric(ctx, r.prefix+"mem_used", fmt.Sprintf("%.2f", metrics.MemUsedMB))
	if err != nil {
		return err
	}

	err = r.setMetric(ctx, r.prefix+"mem_total", fmt.Sprintf("%.2f", metrics.MemTotalMB))
	if err != nil {
		return err
	}

	err = r.setMetric(ctx, r.prefix+"disk_used", fmt.Sprintf("%.2f", metrics.DiskUsed))
	if err != nil {
		return err
	}

	err = r.setMetric(ctx, r.prefix+"disk_total", fmt.Sprintf("%.2f", metrics.DiskTotal))
	if err != nil {
		return err
	}

	err = r.setMetric(ctx, r.prefix+"timestamp", metrics.Timestamp.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisMetricsCache) setMetric(ctx context.Context, key, value string) error {
	_, err := r.client.Set(ctx,
		key,
		value,
		r.ttl).Result()
	if err != nil {
		return fmt.Errorf("не удалось сохранить %v в кэш: %w", key, err)
	}

	return nil
}

// GetMetrics загружает метрики из Redis и рассчитывает производные показатели использования памяти и диска
func (r *RedisMetricsCache) GetMetrics(ctx context.Context) (model.Metrics, error) {
	metricsToReturn := model.Metrics{}

	cpuUsage, err := r.getFloatMetric(ctx, r.prefix+"cpu")
	if err != nil {
		return model.Metrics{}, err
	}
	metricsToReturn.CPUUsage = cpuUsage

	memUsed, err := r.getFloatMetric(ctx, r.prefix+"mem_used")
	if err != nil {
		return model.Metrics{}, err
	}
	metricsToReturn.MemUsedMB = memUsed

	memTotal, err := r.getFloatMetric(ctx, r.prefix+"mem_total")
	if err != nil {
		return model.Metrics{}, err
	}
	metricsToReturn.MemTotalMB = memTotal

	diskUsed, err := r.getFloatMetric(ctx, r.prefix+"disk_used")
	if err != nil {
		return model.Metrics{}, err
	}
	metricsToReturn.DiskUsed = diskUsed

	diskTotal, err := r.getFloatMetric(ctx, r.prefix+"disk_total")
	if err != nil {
		return model.Metrics{}, err
	}
	metricsToReturn.DiskTotal = diskTotal

	ts, err := r.client.Get(ctx, r.prefix+"timestamp").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return model.Metrics{}, system.ErrCacheMiss
		}
		return model.Metrics{}, fmt.Errorf("не удалось получить Timestamp из кэша: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return model.Metrics{}, fmt.Errorf("не удалось распарсить Timestamp: %w", err)
	}

	metricsToReturn.Timestamp = timestamp

	if memTotal == 0 {
		return model.Metrics{}, errors.New("memTotal равен нулю")
	}
	metricsToReturn.MemUsage = memUsed / memTotal * 100

	if diskTotal == 0 {
		return model.Metrics{}, errors.New("diskTotal равен нулю")
	}
	metricsToReturn.DiskUsage = diskUsed / diskTotal * 100

	return metricsToReturn, nil
}

func (r *RedisMetricsCache) getFloatMetric(ctx context.Context, key string) (float64, error) {
	metric, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, system.ErrCacheMiss
		}
		return 0, fmt.Errorf("не удалось получить %v из кэша: %w", key, err)
	}

	metricFloat, err := strconv.ParseFloat(metric, 64)
	if err != nil {
		return 0, fmt.Errorf("не удалось распарсить %v: %w", key, err)
	}

	return metricFloat, nil
}

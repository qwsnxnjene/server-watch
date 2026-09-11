package redis

import (
	"context"
	"errors"
	"fmt"
	"server-watch/internal/system"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("метрики отсутствуют в кэше")

type RedisMetricsCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewRedisMetricsCache(client *redis.Client, ttl time.Duration, prefix string) *RedisMetricsCache {
	return &RedisMetricsCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

func (r *RedisMetricsCache) SetMetrics(metrics system.Metrics) error {
	err := r.setMetric(r.prefix+"cpu", fmt.Sprintf("%.2f", metrics.CPUUsage))
	if err != nil {
		return err
	}

	err = r.setMetric(r.prefix+"mem_used", fmt.Sprintf("%.2f", metrics.MemUsedMB))
	if err != nil {
		return err
	}

	err = r.setMetric(r.prefix+"mem_total", fmt.Sprintf("%.2f", metrics.MemTotalMB))
	if err != nil {
		return err
	}

	err = r.setMetric(r.prefix+"disk_used", fmt.Sprintf("%.2f", metrics.DiskUsed))
	if err != nil {
		return err
	}

	err = r.setMetric(r.prefix+"disk_total", fmt.Sprintf("%.2f", metrics.DiskTotal))
	if err != nil {
		return err
	}

	err = r.setMetric(r.prefix+"timestamp", metrics.Timestamp.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisMetricsCache) setMetric(key, value string) error {
	_, err := r.client.Set(context.Background(),
		key,
		value,
		r.ttl).Result()
	if err != nil {
		return fmt.Errorf("не удалось сохранить %v в кэш: %w", key, err)
	}

	return nil
}

func (r *RedisMetricsCache) GetMetrics() (system.Metrics, error) {
	metricsToReturn := system.Metrics{}

	cpuUsage, err := r.getFloatMetric(r.prefix + "cpu")
	if err != nil {
		return system.Metrics{}, err
	}
	metricsToReturn.CPUUsage = cpuUsage

	memUsed, err := r.getFloatMetric(r.prefix + "mem_used")
	if err != nil {
		return system.Metrics{}, err
	}
	metricsToReturn.MemUsedMB = memUsed

	memTotal, err := r.getFloatMetric(r.prefix + "mem_total")
	if err != nil {
		return system.Metrics{}, err
	}
	metricsToReturn.MemTotalMB = memTotal

	diskUsed, err := r.getFloatMetric(r.prefix + "disk_used")
	if err != nil {
		return system.Metrics{}, err
	}
	metricsToReturn.DiskUsed = diskUsed

	diskTotal, err := r.getFloatMetric(r.prefix + "disk_total")
	if err != nil {
		return system.Metrics{}, err
	}
	metricsToReturn.DiskTotal = diskTotal

	ts, err := r.client.Get(context.Background(), r.prefix+"timestamp").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return system.Metrics{}, ErrCacheMiss
		}
		return system.Metrics{}, fmt.Errorf("не удалось получить Timestamp из кэша: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return system.Metrics{}, fmt.Errorf("не удалось распарсить Timestamp: %w", err)
	}

	metricsToReturn.Timestamp = timestamp

	if memTotal == 0 {
		return system.Metrics{}, errors.New("memTotal равен нулю")
	}
	metricsToReturn.MemUsage = memUsed / memTotal * 100

	if diskTotal == 0 {
		return system.Metrics{}, errors.New("diskTotal равен нулю")
	}
	metricsToReturn.DiskUsage = diskUsed / diskTotal * 100

	return metricsToReturn, nil
}

func (r *RedisMetricsCache) getFloatMetric(key string) (float64, error) {
	metric, err := r.client.Get(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrCacheMiss
		}
		return 0, fmt.Errorf("не удалось получить %v из кэша: %w", key, err)
	}

	metricFloat, err := strconv.ParseFloat(metric, 64)
	if err != nil {
		return 0, fmt.Errorf("не удалось распарсить %v: %w", key, err)
	}

	return metricFloat, nil
}

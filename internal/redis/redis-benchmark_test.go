package redis

import (
	"context"
	"server-watch/internal/system/model"
	"testing"
	"time"
)

func BenchmarkRedisIncrementCount(b *testing.B) {
	client := NewClient()
	store := NewRedisAlertStateStore(client, 5*time.Second, "benchmark:")
	alertType := model.AlertTypeHighCPU
	condition := model.ConditionHigh
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.IncrementCount(ctx, alertType, condition)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRedisIncrementCount_ConditionChanges(b *testing.B) {
	client := NewClient()
	store := NewRedisAlertStateStore(client, 5*time.Second, "benchmark:")
	alertType := model.AlertTypeHighCPU
	conditions := []model.AlertCondition{model.ConditionHigh, model.ConditionNormal}
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.IncrementCount(ctx, alertType, conditions[i%2])
		if err != nil {
			b.Fatal(err)
		}
	}
}

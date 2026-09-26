package alert_state_store

import (
	"context"
	"server-watch/internal/system/model"
	"testing"
)

func BenchmarkIncrementCount(b *testing.B) {
	store := NewInMemoryAlertStateStore()
	alertType := model.AlertTypeHighCPU
	condition := model.ConditionHigh
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.IncrementCount(ctx, alertType, condition)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIncrementCount_ConditionChanges(b *testing.B) {
	store := NewInMemoryAlertStateStore()
	alertType := model.AlertTypeHighCPU
	conditions := []model.AlertCondition{model.ConditionHigh, model.ConditionNormal}
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.IncrementCount(ctx, alertType, conditions[i%2])
		if err != nil {
			b.Fatal(err)
		}
	}
}

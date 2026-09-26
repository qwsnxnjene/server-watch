package storage

import (
	"context"
	"server-watch/internal/system/model"
	"testing"
	"time"
)

func BenchmarkGetMetrics(b *testing.B) {
	db := newTestDB(b)
	repo := NewSQLiteRepository(db)
	now := time.Now()
	from := now.Add(-time.Minute)
	to := now.Add(time.Minute)

	metrics := model.Metrics{
		CPUUsage:  35.2,
		MemUsage:  62.1,
		MemUsedMB:   4096,
		MemTotalMB:  8192,
		DiskUsage: 48.7,
		DiskUsed:  120,
		DiskTotal: 250,
		Timestamp:  now,
	}
	ctx := context.Background()

	for range 100 {
		if err := repo.SaveMetrics(ctx, metrics); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := repo.GetMetrics(ctx, from, to); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSaveMetrics(b *testing.B) {
	db := newTestDB(b)
	repo := NewSQLiteRepository(db)
	now := time.Now()
	metrics := model.Metrics{
		CPUUsage:  35.2,
		MemUsage:  62.1,
		MemUsedMB:   4096,
		MemTotalMB:  8192,
		DiskUsage: 48.7,
		DiskUsed:  120,
		DiskTotal: 250,
		Timestamp:  now,
	}
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := repo.SaveMetrics(ctx, metrics); err != nil {
			b.Fatal(err)
		}
	}
}

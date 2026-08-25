package storage

import (
	"database/sql"
	"fmt"
	"server-watch/internal/system"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

func (s *SQLiteRepository) SaveMetrics(metrics system.Metrics) error {
	_, err := s.db.Exec(`
    INSERT INTO metrics (
        ts,
        cpu,
        mem_used_mb,
        mem_total_mb,
        disk_used_gb,
        disk_total_gb
    ) VALUES (
        :ts,
        :cpu,
        :mem_used_mb,
        :mem_total_mb,
        :disk_used_gb,
        :disk_total_gb
    )
	`,
		sql.Named("ts", metrics.Timestamp.UTC()),
		sql.Named("cpu", metrics.CPUUsage),
		sql.Named("mem_used_mb", int64(metrics.MemUsedMB)),
		sql.Named("mem_total_mb", int64(metrics.MemTotalMB)),
		sql.Named("disk_used_gb", int64(metrics.DiskUsed)),
		sql.Named("disk_total_gb", int64(metrics.DiskTotal)),
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения метрик в базу данных SQLite: %w", err)
	}

	return nil
}

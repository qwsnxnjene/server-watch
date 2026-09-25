package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server-watch/internal/system/model"
	"time"
)

// SQLiteRepository реализует работу с метриками и алертами через SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository создаёт репозиторий на основе SQLite-соединения
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// SaveMetrics сохраняет измерение системных метрик в SQLite
func (s *SQLiteRepository) SaveMetrics(ctx context.Context, metrics model.Metrics) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
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
		return fmt.Errorf("не удалось сохранить метрики в базу данных SQLite: %w", err)
	}

	return nil
}

// SaveAlert сохраняет алерт в SQLite и возвращает его идентификатор
func (s *SQLiteRepository) SaveAlert(ctx context.Context, alert model.Alert) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	res, err := s.db.ExecContext(ctx, `
	INSERT INTO alerts (
	    ts,
	    type,
	    threshold,
	    value,
	    resolved,
	    resolved_ts
	) VALUES (
		:ts,
	    :type,
	    :threshold,
	    :value,
	    :resolved,
	    :resolved_ts
	)
	`,
		sql.Named("ts", alert.Timestamp),
		sql.Named("type", alert.Type),
		sql.Named("threshold", alert.Threshold),
		sql.Named("value", alert.Value),
		sql.Named("resolved", alert.Resolved),
		sql.Named("resolved_ts", alert.ResolvedAt),
	)

	if err != nil {
		return 0, fmt.Errorf("не удалось сохранить алерт в базу данных SQLite: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("не удалось получить id добавленного алерта: %w", err)
	}

	return id, nil
}

// GetMetrics возвращает измерения метрик за указанный временной диапазон
func (s *SQLiteRepository) GetMetrics(ctx context.Context, from, to time.Time) ([]model.Metrics, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
	SELECT ts, cpu, mem_used_mb, mem_total_mb, disk_used_gb, disk_total_gb
	FROM metrics
	WHERE ts BETWEEN :from AND :to
	`,
		sql.Named("from", from),
		sql.Named("to", to),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("не удалось получить список метрик из базы данных SQLite: %w", err)
	}
	defer rows.Close()

	var metrics []model.Metrics

	for rows.Next() {
		var ts time.Time
		var cpu, memUsedMb, memTotalMb, diskUsedGb, diskTotalGb float64

		err := rows.Scan(&ts, &cpu, &memUsedMb, &memTotalMb, &diskUsedGb, &diskTotalGb)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать метрику из базы данных SQLite: %w", err)
		}

		currMetrics := model.Metrics{
			Timestamp:  ts,
			CPUUsage:   cpu,
			MemUsedMB:  memUsedMb,
			MemTotalMB: memTotalMb,
			DiskUsed:   diskUsedGb,
			DiskTotal:  diskTotalGb,
			MemUsage:   memUsedMb / memTotalMb * 100.0,
			DiskUsage:  diskUsedGb / diskTotalGb * 100.0,
		}

		metrics = append(metrics, currMetrics)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать метрику из базы данных SQLite: %w", err)
	}

	return metrics, nil
}

// GetAlerts возвращает сохранённые алерты.
// При activeOnly == true возвращаются только активные алерты
func (s *SQLiteRepository) GetAlerts(ctx context.Context, activeOnly bool) ([]model.Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	query := `
	SELECT id, ts, type, threshold, value, resolved, resolved_ts
	FROM alerts
`

	if activeOnly {
		query += ` WHERE resolved = FALSE`
	}

	rows, err := s.db.QueryContext(ctx, query)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("не удалось получить список алертов из базы данных SQLite: %w", err)
	}
	defer rows.Close()

	var alerts []model.Alert

	for rows.Next() {
		var id int64
		var ts time.Time
		var alertType string
		var resolved bool
		var threshold, value float64
		var resolvedTs sql.NullTime

		err := rows.Scan(&id, &ts, &alertType, &threshold, &value, &resolved, &resolvedTs)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать алерт из базы данных SQLite: %w", err)
		}

		var resolvedAt *time.Time
		if resolvedTs.Valid {
			resolvedAt = &resolvedTs.Time
		}

		currAlert := model.Alert{
			ID:         id,
			Type:       model.AlertType(alertType),
			Timestamp:  ts,
			Threshold:  threshold,
			Resolved:   resolved,
			ResolvedAt: resolvedAt,
			Value:      value,
		}

		alerts = append(alerts, currAlert)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать алерт из базы данных SQLite: %w", err)
	}

	return alerts, nil
}

// ResolveAlert помечает алерт как разрешённый и сохраняет время разрешения
func (s *SQLiteRepository) ResolveAlert(ctx context.Context, id int64, resolvedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
	UPDATE alerts
	SET resolved = TRUE, resolved_ts = :resolvedTs
	WHERE id = :id
`,
		sql.Named("resolvedTs", resolvedAt),
		sql.Named("id", id))

	if err != nil {
		return fmt.Errorf("не удалось зарезолвить алерт: %w", err)
	}

	return nil
}

// GetActiveAlert возвращает активный алерт указанного типа.
// Если активного алерта нет, возвращается nil без ошибки
func (s *SQLiteRepository) GetActiveAlert(ctx context.Context, alertType model.AlertType) (*model.Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	row := s.db.QueryRowContext(ctx, `
	SELECT id, ts, type, threshold, value, resolved, resolved_ts
	FROM alerts
	WHERE type = :type AND resolved = FALSE
`,
		sql.Named("type", alertType))

	var id int64
	var ts time.Time
	var typeAlert string
	var resolved bool
	var threshold, value float64
	var resolvedTs sql.NullTime

	if err := row.Scan(&id, &ts, &typeAlert, &threshold, &value, &resolved, &resolvedTs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("не удалось получить действующий алерт из базы данных SQLite: %w", err)
	}

	var resolvedAt *time.Time
	if resolvedTs.Valid {
		resolvedAt = &resolvedTs.Time
	}

	alert := model.Alert{
		ID:         id,
		Type:       model.AlertType(typeAlert),
		Timestamp:  ts,
		Threshold:  threshold,
		Resolved:   resolved,
		ResolvedAt: resolvedAt,
		Value:      value,
	}

	return &alert, nil
}

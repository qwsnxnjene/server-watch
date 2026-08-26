package storage

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"server-watch/internal/system"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("не удалось подключиться к тестовой базе данных: %v", err)
	}

	db.SetMaxOpenConns(1)

	err = Migrate(db)
	if err != nil {
		t.Fatalf("не удалось сделать миграцию тестовой базы данных: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestSQLiteRepository_SaveMetrics(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	metrics := system.Metrics{
		CPUUsage:   25.5,
		MemUsage:   40.0,
		MemUsedMB:  4000,
		MemTotalMB: 10000,
		DiskUsage:  50.0,
		DiskUsed:   50,
		DiskTotal:  100,
		Timestamp:  time.Now().UTC(),
	}

	err := repo.SaveMetrics(metrics)
	if err != nil {
		t.Fatalf("не удалось сохранить метрики: %v", err)
	}

	from := metrics.Timestamp.Add(-time.Minute)
	to := metrics.Timestamp.Add(time.Minute)

	metricsGot, err := repo.GetMetrics(from, to)
	if err != nil {
		t.Fatalf("не удалось получить сохраненные метрики из базы данных: %v", err)
	}
	if metricsGot == nil {
		t.Fatal("сохраненные метрики не найдены")
	}

	if len(metricsGot) != 1 {
		t.Fatalf("ожидалась 1 метрика, получено: %d", len(metricsGot))
	}

	if metrics.CPUUsage != metricsGot[0].CPUUsage {
		t.Fatalf("CPUUsage метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.CPUUsage, metricsGot[0].CPUUsage)
	}
	if metrics.MemUsage != metricsGot[0].MemUsage {
		t.Fatalf("MemUsage метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.MemUsage, metricsGot[0].MemUsage)
	}
	if metrics.MemUsedMB != metricsGot[0].MemUsedMB {
		t.Fatalf("MemUsedMB метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.MemUsedMB, metricsGot[0].MemUsedMB)
	}
	if metrics.MemTotalMB != metricsGot[0].MemTotalMB {
		t.Fatalf("MemTotalMB метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.MemTotalMB, metricsGot[0].MemTotalMB)
	}
	if metrics.DiskUsage != metricsGot[0].DiskUsage {
		t.Fatalf("DiskUsage метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.DiskUsage, metricsGot[0].DiskUsage)
	}
	if metrics.DiskUsed != metricsGot[0].DiskUsed {
		t.Fatalf("DiskUsed метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.DiskUsed, metricsGot[0].DiskUsed)
	}
	if metrics.DiskTotal != metricsGot[0].DiskTotal {
		t.Fatalf("DiskTotal метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.DiskTotal, metricsGot[0].DiskTotal)
	}
	if !metrics.Timestamp.Equal(metricsGot[0].Timestamp) {
		t.Fatalf("timestamps метрик не совпадают:\nожидали: %v\nполучили: %v",
			metrics.Timestamp, metricsGot[0].Timestamp)
	}
}

func TestSQLiteRepository_SaveAlert(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	alert := system.Alert{
		Type:       system.AlertTypeHighCPU,
		Timestamp:  time.Now(),
		Threshold:  32.5,
		Resolved:   false,
		ResolvedAt: nil,
		Value:      23.2,
	}

	id, err := repo.SaveAlert(alert)
	if err != nil {
		t.Fatalf("не удалось сохранить алерт в базу данных: %v", err)
	}
	alert.ID = id

	alertGot, err := repo.GetActiveAlert(system.AlertTypeHighCPU)
	if err != nil {
		t.Fatalf("не удалось получить алерт из базы данных: %v", err)
	}
	if alertGot == nil {
		t.Fatal("не получено алертов из базы данных по запросу")
	}

	if alertGot.ID != alert.ID {
		t.Fatalf("ID алертов не совпадают:\nожидали: %v\nполучили: %v", alert.ID, alertGot.ID)
	}
	if alertGot.Type != alert.Type {
		t.Fatalf("типы алертов не совпадают:\nожидали: %v\nполучили: %v", alert.Type, alertGot.Type)
	}
	if alertGot.Threshold != alert.Threshold {
		t.Fatalf("threshold алертов не совпадают:\nожидали: %v\nполучили: %v", alert.Threshold, alertGot.Threshold)
	}
	if alertGot.Value != alert.Value {
		t.Fatalf("значение алертов не совпадают:\nожидали: %v\nполучили: %v", alert.Value, alertGot.Value)
	}
	if alertGot.Resolved != alert.Resolved {
		t.Fatalf("статусы алертов не совпадают:\nожидали: %v\nполучили: %v", alert.Resolved, alertGot.Resolved)
	}
	if !alertGot.Timestamp.Equal(alert.Timestamp) {
		t.Fatalf("timestamps алертов не совпадают:\nожидали: %v\nполучили: %v", alert.Timestamp, alertGot.Timestamp)
	}
}

func TestSQLiteRepository_ResolveAlert(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	alert := system.Alert{
		Type:       system.AlertTypeHighCPU,
		Timestamp:  time.Now(),
		Threshold:  32.5,
		Resolved:   false,
		ResolvedAt: nil,
		Value:      23.2,
	}

	id, err := repo.SaveAlert(alert)
	if err != nil {
		t.Fatalf("не удалось сохранить алерт в базу данных: %v", err)
	}
	alert.ID = id

	alertGot, err := repo.GetActiveAlert(system.AlertTypeHighCPU)
	if err != nil {
		t.Fatalf("не удалось получить сохраненный алерт из базы данных: %v", err)
	}
	if alertGot == nil {
		t.Fatal("сохраненный алерт не найден")
	}

	err = repo.ResolveAlert(id, time.Now())
	if err != nil {
		t.Fatalf("не удалось зарезолвить алерт: %v", err)
	}

	alertGot, err = repo.GetActiveAlert(system.AlertTypeHighCPU)
	if err != nil {
		t.Fatalf("не удалось получить алерт из базы данных: %v", err)
	}
	if alertGot != nil {
		t.Fatal("алерт до сих пор считается активным после резолва")
	}
}

func TestSQLiteRepository_GetAlerts(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	alert1 := system.Alert{
		Type:       system.AlertTypeHighCPU,
		Timestamp:  time.Now(),
		Threshold:  32.5,
		Resolved:   false,
		ResolvedAt: nil,
		Value:      23.2,
	}

	alert2 := system.Alert{
		Type:       system.AlertTypeHighMem,
		Timestamp:  time.Now(),
		Threshold:  45.2,
		Resolved:   false,
		ResolvedAt: nil,
		Value:      13.0,
	}

	//сохраняем оба алерта в базу данных
	id1, err := repo.SaveAlert(alert1)
	if err != nil {
		t.Fatalf("не удалось сохранить алерт в базу данных: %v", err)
	}
	alert1.ID = id1

	id2, err := repo.SaveAlert(alert2)
	if err != nil {
		t.Fatalf("не удалось сохранить алерт в базу данных: %v", err)
	}
	alert2.ID = id2

	//отмечаем один из алертов как завершенный
	err = repo.ResolveAlert(id1, time.Now())
	if err != nil {
		t.Fatalf("не удалось зарезолвить алерт: %v", err)
	}

	//получаем все алерты
	alertsGot, err := repo.GetAlerts(false)
	if err != nil {
		t.Fatalf("не удалось получить все сохраненные алерты: %v", err)
	}
	if alertsGot == nil {
		t.Fatal("сохраненные алерты не найдены")
	}

	if len(alertsGot) != 2 {
		t.Fatalf("ожидалось 2 алерта, получено %v", len(alertsGot))
	}

	//получаем только активные алерты
	activeAlertsGot, err := repo.GetAlerts(true)
	if err != nil {
		t.Fatalf("не удалось получить активные сохраненные алерты: %v", err)
	}

	if len(activeAlertsGot) != 1 {
		t.Fatalf("ожидался 1 активный алерт, получено %v", len(activeAlertsGot))
	}

	if activeAlertsGot[0].ID != alert2.ID {
		t.Fatalf(
			"получен неправильный активный алерт: ожидался ID %d, получен %d",
			alert2.ID,
			activeAlertsGot[0].ID,
		)
	}
}

func TestSQLiteRepository_GetMetrics(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	now := time.Now().UTC()

	metrics1 := system.Metrics{
		CPUUsage:   25.5,
		MemUsage:   40.0,
		MemUsedMB:  4000,
		MemTotalMB: 10000,
		DiskUsage:  50.0,
		DiskUsed:   50,
		DiskTotal:  100,
		Timestamp:  now.Add(-10 * time.Minute),
	}

	metrics2 := system.Metrics{
		CPUUsage:   75.2,
		MemUsage:   75.0,
		MemUsedMB:  6000,
		MemTotalMB: 8000,
		DiskUsage:  60.0,
		DiskUsed:   60,
		DiskTotal:  100,
		Timestamp:  now.Add(-5 * time.Minute),
	}

	err := repo.SaveMetrics(metrics1)
	if err != nil {
		t.Fatalf("не удалось сохранить метрику: %v", err)
	}

	err = repo.SaveMetrics(metrics2)
	if err != nil {
		t.Fatalf("не удалось сохранить метрику: %v", err)
	}

	from := now.Add(-7 * time.Minute)
	to := now

	metrics, err := repo.GetMetrics(from, to)
	if err != nil {
		t.Fatalf("не удалось получить сохраненные метрики: %v", err)
	}
	if metrics == nil {
		t.Fatal("сохраненные метрики в заданном временном промежутке не найдены")
	}

	if len(metrics) != 1 {
		t.Fatalf("ожидалась 1 сохраненная метрика в заданном временном промежутке, получено %v",
			len(metrics))
	}

	if !metrics[0].Timestamp.Equal(metrics2.Timestamp) {
		t.Fatalf("метрики не совпадают по timestamp:\nожидали: %v\n, получили: %v",
			metrics2.Timestamp, metrics[0].Timestamp)
	}
}

func TestSQLiteRepository_GetMetrics_IncludesBoundaries(t *testing.T) {
	db := newTestDB(t)
	repo := NewSQLiteRepository(db)

	now := time.Now().UTC()

	metrics1 := system.Metrics{
		CPUUsage:   25.5,
		MemUsage:   40.0,
		MemUsedMB:  4000,
		MemTotalMB: 10000,
		DiskUsage:  50.0,
		DiskUsed:   50,
		DiskTotal:  100,
		Timestamp:  now.Add(-10 * time.Minute),
	}

	metrics2 := system.Metrics{
		CPUUsage:   75.2,
		MemUsage:   75.0,
		MemUsedMB:  6000,
		MemTotalMB: 8000,
		DiskUsage:  60.0,
		DiskUsed:   60,
		DiskTotal:  100,
		Timestamp:  now.Add(-5 * time.Minute),
	}

	err := repo.SaveMetrics(metrics1)
	if err != nil {
		t.Fatalf("не удалось сохранить метрику: %v", err)
	}

	err = repo.SaveMetrics(metrics2)
	if err != nil {
		t.Fatalf("не удалось сохранить метрику: %v", err)
	}

	from := now.Add(-5 * time.Minute)
	to := now

	metrics, err := repo.GetMetrics(from, to)
	if err != nil {
		t.Fatalf("не удалось получить сохраненные метрики: %v", err)
	}
	if metrics == nil {
		t.Fatal("сохраненные метрики в заданном временном промежутке не найдены")
	}

	if len(metrics) != 1 {
		t.Fatalf("ожидалась 1 сохраненная метрика в заданном временном промежутке, получено %v",
			len(metrics))
	}

	if !metrics[0].Timestamp.Equal(metrics2.Timestamp) {
		t.Fatalf("метрики не совпадают по timestamp:\nожидали: %v\n, получили: %v",
			metrics2.Timestamp, metrics[0].Timestamp)
	}
}

package system

import (
	"errors"
	"fmt"
	"path/filepath"
	"server-watch/internal/config"
	"server-watch/internal/system/model"
	"testing"
	"time"
)

type FakeRepository struct {
	metrics []Metrics
	alerts  []model.Alert

	saveMetricsErr    error
	saveAlertErr      error
	getActiveAlertErr error
	resolveAlertErr   error
	getMetricsErr     error
	getAlertsErr      error

	nextAlertID int64
}

func (f *FakeRepository) SaveMetrics(metrics Metrics) error {
	if f.saveMetricsErr != nil {
		return f.saveMetricsErr
	}

	f.metrics = append(f.metrics, metrics)
	return nil
}

func (f *FakeRepository) SaveAlert(alert model.Alert) (int64, error) {
	if f.saveAlertErr != nil {
		return 0, f.saveAlertErr
	}

	alert.ID = f.nextAlertID
	f.alerts = append(f.alerts, alert)
	f.nextAlertID++
	return alert.ID, nil
}

func (f *FakeRepository) GetMetrics(from time.Time, to time.Time) ([]Metrics, error) {
	if f.getMetricsErr != nil {
		return nil, f.getMetricsErr
	}

	var metrics []Metrics
	for _, metric := range f.metrics {
		if !metric.Timestamp.After(to) && !metric.Timestamp.Before(from) {
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}

func (f *FakeRepository) GetAlerts(activeOnly bool) ([]model.Alert, error) {
	if f.getAlertsErr != nil {
		return nil, f.getAlertsErr
	}

	if !activeOnly {
		return f.alerts, nil
	}

	var alerts []model.Alert
	for _, alert := range f.alerts {
		if !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

func (f *FakeRepository) ResolveAlert(id int64, resolvedAt time.Time) error {
	if f.resolveAlertErr != nil {
		return f.resolveAlertErr
	}

	for i, alert := range f.alerts {
		if alert.ID == id {
			f.alerts[i].Resolved = true
			f.alerts[i].ResolvedAt = &resolvedAt
			return nil
		}
	}

	return errors.New("не удалось найти алерт с таким ID")
}

func (f *FakeRepository) GetActiveAlert(alertType model.AlertType) (*model.Alert, error) {
	if f.getActiveAlertErr != nil {
		return nil, f.getActiveAlertErr
	}

	for _, alert := range f.alerts {
		if alert.Type == alertType {
			return &alert, nil
		}
	}

	return nil, nil
}

type FakeMetricsCache struct {
	metrics Metrics
	getErr  error
	setErr  error
}

func (f *FakeMetricsCache) SetMetrics(metrics Metrics) error {
	f.metrics = metrics
	return f.setErr
}

func (f *FakeMetricsCache) GetMetrics() (Metrics, error) {
	return f.metrics, f.getErr
}

func newTestSystem(fakeRepo *FakeRepository, fakeCache *FakeMetricsCache) *System {
	return NewSystem(
		fakeRepo,
		NewFallbackAlertStateStore(&mockAlertStateStore{}, &mockAlertStateStore{}),
		config.Config{},
		"",
		fakeCache)
}

func TestSystem_CollectMetrics(t *testing.T) {
	fakeErr := fmt.Errorf("database unavailable")

	tests := []struct {
		name     string
		repoErr  error
		cacheErr error
		wantErr  error
	}{
		{
			name:    "успешное сохранение",
			repoErr: nil,
			wantErr: nil,
		},
		{
			name:    "ошибка сохранения",
			repoErr: fakeErr,
			wantErr: fakeErr,
		},
		{
			name:     "ошибка кэша",
			cacheErr: ErrCacheMiss,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepository{
				metrics:        make([]Metrics, 0),
				saveMetricsErr: tt.repoErr,
			}

			system := newTestSystem(fakeRepo, &FakeMetricsCache{setErr: tt.cacheErr})

			err := system.CollectMetrics()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
			}
		})
	}
}

func TestSystem_ProcessAlerts_CreateAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]model.Alert, 0)}
	system := newTestSystem(fakeRepo, &FakeMetricsCache{})

	metrics := Metrics{
		CPUUsage: 95.5,
	}
	update := model.AlertUpdate{
		CPUCount:     AlertTriggerCount,
		CPUCondition: model.ConditionHigh,
	}

	err := system.processAlerts(metrics, update)
	if err != nil {
		t.Fatalf("ошибка обработки алерта: %v", err)
	}

	if len(fakeRepo.alerts) != 1 {
		t.Fatalf("ожидался 1 созданный алерт, получили %v", len(fakeRepo.alerts))
	}

	alert := fakeRepo.alerts[0]

	if alert.Type != model.AlertTypeHighCPU {
		t.Fatalf("ожидали тип алерта %v, получили %v", model.AlertTypeHighCPU, alert.Type)
	}
	if alert.Threshold != HighCPUThreshold {
		t.Fatalf("ожидали порог = %v, получили %v", HighCPUThreshold, alert.Threshold)
	}
	if alert.Value != metrics.CPUUsage {
		t.Fatalf("ожидали значение алерта = %v, получили %v", metrics.CPUUsage, alert.Value)
	}
	if alert.Resolved {
		t.Fatalf("алерт зарезолвлен, хотя не должен быть")
	}
	if alert.ResolvedAt != nil {
		t.Fatalf("время резолва алерта != nil")
	}
	if alert.Timestamp.IsZero() {
		t.Fatal("у созданного алерта не установлен timestamp")
	}
}

func TestSystem_ProcessAlerts_CreateAlertWithActive(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]model.Alert, 0)}

	// сохраняем активный алерт до теста, чтобы новый алерт не создавался
	_, err := fakeRepo.SaveAlert(model.Alert{
		Type: model.AlertTypeHighCPU,
	})
	if err != nil {
		t.Fatalf("не удалось сохранить алерт: %v", err)
	}
	system := newTestSystem(fakeRepo, &FakeMetricsCache{})

	metrics := Metrics{
		CPUUsage: 95.5,
	}
	update := model.AlertUpdate{
		CPUCount:     AlertTriggerCount,
		CPUCondition: model.ConditionHigh,
	}

	err = system.processAlerts(metrics, update)
	if err != nil {
		t.Fatalf("ошибка обработки алерта: %v", err)
	}

	if len(fakeRepo.alerts) != 1 {
		t.Fatalf("ожидался 1 созданный алерт, получили %v", len(fakeRepo.alerts))
	}
}

func TestSystem_ProcessAlerts_CreateAlert_GetActiveAlertError(t *testing.T) {
	fakeErr := errors.New("не удалось получить список активных алертов")

	fakeRepo := &FakeRepository{
		metrics:           make([]Metrics, 0),
		alerts:            make([]model.Alert, 0),
		getActiveAlertErr: fakeErr,
	}
	system := newTestSystem(fakeRepo, &FakeMetricsCache{})

	metrics := Metrics{
		CPUUsage: 95.5,
	}
	update := model.AlertUpdate{
		CPUCount:     AlertTriggerCount,
		CPUCondition: model.ConditionHigh,
	}

	err := system.processAlerts(metrics, update)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_CreateAlert_SaveAlertError(t *testing.T) {
	fakeErr := errors.New("не удалось сохранить алерт")

	fakeRepo := &FakeRepository{
		metrics:      make([]Metrics, 0),
		alerts:       make([]model.Alert, 0),
		saveAlertErr: fakeErr,
	}
	system := newTestSystem(fakeRepo, &FakeMetricsCache{})

	metrics := Metrics{
		CPUUsage: 95.5,
	}
	update := model.AlertUpdate{
		CPUCount:     AlertTriggerCount,
		CPUCondition: model.ConditionHigh,
	}

	err := system.processAlerts(metrics, update)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]model.Alert, 0)}

	alert := model.Alert{
		ID:         1,
		Type:       model.AlertTypeHighCPU,
		Resolved:   false,
		ResolvedAt: nil,
	}

	fakeRepo.alerts = append(fakeRepo.alerts, alert)

	system := newTestSystem(fakeRepo, &FakeMetricsCache{})
	update := model.AlertUpdate{
		CPUCount:     AlertResolveCount,
		CPUCondition: model.ConditionNormal,
	}

	err := system.processAlerts(Metrics{}, update)
	if err != nil {
		t.Fatalf("не удалось обработать алерт: %v", err)
	}

	if !fakeRepo.alerts[0].Resolved {
		t.Fatal("алерт не был зарезолвлен")
	}
	if fakeRepo.alerts[0].ResolvedAt == nil {
		t.Fatal("время резолва не установлено")
	}
}

func TestSystem_ProcessAlerts_ResolveAlert_GetActiveAlertError(t *testing.T) {
	fakeErr := errors.New("не удалось получить активные алерты")

	fakeRepo := &FakeRepository{
		metrics:           make([]Metrics, 0),
		alerts:            make([]model.Alert, 0),
		getActiveAlertErr: fakeErr,
	}

	system := newTestSystem(fakeRepo, &FakeMetricsCache{})
	update := model.AlertUpdate{
		CPUCount:     AlertResolveCount,
		CPUCondition: model.ConditionNormal,
	}

	err := system.processAlerts(Metrics{}, update)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert_ResolveError(t *testing.T) {
	fakeErr := errors.New("не удалось зарезолвить алерт")

	fakeRepo := &FakeRepository{
		metrics:         make([]Metrics, 0),
		alerts:          make([]model.Alert, 0),
		resolveAlertErr: fakeErr,
	}

	alert := model.Alert{
		ID:         1,
		Type:       model.AlertTypeHighCPU,
		Resolved:   false,
		ResolvedAt: nil,
	}

	fakeRepo.alerts = append(fakeRepo.alerts, alert)

	system := newTestSystem(fakeRepo, &FakeMetricsCache{})
	update := model.AlertUpdate{
		CPUCount:     AlertResolveCount,
		CPUCondition: model.ConditionNormal,
	}

	err := system.processAlerts(Metrics{}, update)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert_NoActiveAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]model.Alert, 0)}

	system := newTestSystem(fakeRepo, &FakeMetricsCache{})
	update := model.AlertUpdate{
		CPUCount:     AlertResolveCount,
		CPUCondition: model.ConditionNormal,
	}

	err := system.processAlerts(Metrics{}, update)
	if err != nil {
		t.Fatalf("не удалось обработать алерт: %v", err)
	}
}

func TestSystem_GetHistory(t *testing.T) {
	fakeRepo := FakeRepository{
		metrics: make([]Metrics, 0),
	}

	now := time.Now().UTC()

	metricsToAdd := []Metrics{{Timestamp: now}, {Timestamp: now}, {Timestamp: now.Add(-time.Minute * 5)}}
	for _, metric := range metricsToAdd {
		err := fakeRepo.SaveMetrics(metric)
		if err != nil {
			t.Fatalf("не удалось добавить метрику: %v", err)
		}
	}

	system := newTestSystem(&fakeRepo, &FakeMetricsCache{})
	metrics, err := system.GetHistory(now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("не удалось получить историю измерений метрик: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("ожидалось 2 метрики, получили %v", len(metrics))
	}
}

func TestSystem_GetHistory_GetMetricsError(t *testing.T) {
	fakeErr := errors.New("не удалось получить список измерений метрик")

	fakeRepo := FakeRepository{
		metrics:       make([]Metrics, 0),
		getMetricsErr: fakeErr,
	}

	now := time.Now().UTC()

	system := newTestSystem(&fakeRepo, &FakeMetricsCache{})
	_, err := system.GetHistory(now.Add(-time.Minute), now.Add(time.Minute))
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_GetHistory_IncludeBoundaries(t *testing.T) {
	fakeRepo := FakeRepository{
		metrics: make([]Metrics, 0),
	}

	now := time.Now().UTC()

	metricsToAdd := []Metrics{
		{Timestamp: now},
		{Timestamp: now.Add(-time.Minute)},
		{Timestamp: now.Add(-time.Minute - time.Second)}}
	for _, metric := range metricsToAdd {
		err := fakeRepo.SaveMetrics(metric)
		if err != nil {
			t.Fatalf("не удалось добавить метрику: %v", err)
		}
	}

	system := newTestSystem(&fakeRepo, &FakeMetricsCache{})
	metrics, err := system.GetHistory(now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("не удалось получить историю измерений метрик: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("ожидалось 2 метрики, получили %v", len(metrics))
	}
}

func TestSystem_GetAlerts(t *testing.T) {
	tests := []struct {
		name        string
		alertsToAdd []model.Alert
		wantAlerts  int
		activeOnly  bool
	}{
		{
			name: "все алерты",
			alertsToAdd: []model.Alert{
				{Type: model.AlertTypeHighCPU},
				{Type: model.AlertTypeHighMem},
				{Type: model.AlertTypeHighCPU, Threshold: HighCPUThreshold, Resolved: true},
			},
			wantAlerts: 3,
			activeOnly: false,
		},
		{
			name: "только активные алерты",
			alertsToAdd: []model.Alert{
				{Type: model.AlertTypeHighCPU},
				{Type: model.AlertTypeHighMem},
				{Type: model.AlertTypeHighCPU, Threshold: HighCPUThreshold, Resolved: true},
			},
			wantAlerts: 2,
			activeOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := FakeRepository{alerts: make([]model.Alert, 0)}

			for i := range tt.alertsToAdd {
				_, err := fakeRepo.SaveAlert(tt.alertsToAdd[i])
				if err != nil {
					t.Fatalf("не удалось добавить алерт: %v", err)
				}
			}

			system := newTestSystem(&fakeRepo, &FakeMetricsCache{})

			alerts, err := system.GetAlerts(tt.activeOnly)
			if err != nil {
				t.Fatalf("не удалось получить список алертов: %v", err)
			}

			if len(alerts) != tt.wantAlerts {
				t.Fatalf("ожидали %v алертов, получили %v", tt.wantAlerts, len(alerts))
			}
		})
	}
}

func TestSystem_GetAlerts_Error(t *testing.T) {
	fakeErr := errors.New("не удалось получить список алертов")

	fakeRepo := FakeRepository{
		alerts:       make([]model.Alert, 0),
		getAlertsErr: fakeErr,
	}

	system := newTestSystem(&fakeRepo, &FakeMetricsCache{})

	_, err := system.GetAlerts(true)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_UpdateConfig(t *testing.T) {
	fakeRepo := FakeRepository{}
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	initialCfg := config.DefaultConfig()
	sys := NewSystem(
		&fakeRepo,
		NewFallbackAlertStateStore(&mockAlertStateStore{}, &mockAlertStateStore{}),
		initialCfg,
		configPath,
		&FakeMetricsCache{})

	cpuCorrect := 45.0

	updatedCfg := ConfigUpdate{
		CPUThreshold: &cpuCorrect,
	}

	want := config.Config{
		CPUThreshold: cpuCorrect,
		MemThreshold: 90,
		TriggerCount: 3,
		ResolveCount: 3,
		SlackEnabled: false,
		SlackURL:     "",
	}

	err := sys.UpdateConfig(updatedCfg)
	if err != nil {
		t.Fatalf("не удалось обновить конфигурацию: %v", err)
	}

	if want != sys.config {
		t.Fatalf("ожидали %+v, получили %+v", want, sys.config)
	}

	gotFromFile, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("не удалось прочитать конфигурацию из файла: %v", err)
	}

	if want != gotFromFile {
		t.Fatalf("ожидали в YAML %+v, получили %+v", want, gotFromFile)
	}
}

func TestSystem_UpdateConfig_InvalidConfig(t *testing.T) {
	fakeRepo := FakeRepository{}
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	initialCfg := config.DefaultConfig()
	sys := NewSystem(
		&fakeRepo,
		NewFallbackAlertStateStore(&mockAlertStateStore{}, &mockAlertStateStore{}),
		initialCfg,
		configPath,
		&FakeMetricsCache{},
	)

	cpuInvalid := 135.0

	updatedCfg := ConfigUpdate{
		CPUThreshold: &cpuInvalid,
	}

	want := config.Config{
		CPUThreshold: 80,
		MemThreshold: 90,
		TriggerCount: 3,
		ResolveCount: 3,
		SlackEnabled: false,
		SlackURL:     "",
	}

	err := sys.UpdateConfig(updatedCfg)
	if !errors.Is(err, config.ErrInvalidConfig) {
		t.Fatalf("ожидали ошибку %v, получили ошибку %v", config.ErrInvalidConfig, err)
	}

	if want != sys.config {
		t.Fatalf("ожидали %+v, получили %+v", want, sys.config)
	}
}

func TestSystem_UpdateConfig_SaveError(t *testing.T) {
	fakeRepo := FakeRepository{}

	configPath := t.TempDir()

	initialCfg := config.DefaultConfig()
	sys := NewSystem(
		&fakeRepo,
		NewFallbackAlertStateStore(&mockAlertStateStore{}, &mockAlertStateStore{}),
		initialCfg,
		configPath,
		&FakeMetricsCache{})

	cpuNew := 70.0

	updatedCfg := ConfigUpdate{
		CPUThreshold: &cpuNew,
	}

	err := sys.UpdateConfig(updatedCfg)
	if err == nil {
		t.Fatal("ожидали ошибку сохранения конфигурации")
	}

	want := config.Config{
		CPUThreshold: 70,
		MemThreshold: 90,
		TriggerCount: 3,
		ResolveCount: 3,
		SlackEnabled: false,
		SlackURL:     "",
	}

	if want != sys.config {
		t.Fatalf("ожидали %+v, получили %+v", want, sys.config)
	}
}

func TestSystem_GetMetrics_FromCache(t *testing.T) {
	fakeCache := &FakeMetricsCache{metrics: Metrics{CPUUsage: 50.0}}

	fakeSys := newTestSystem(nil, fakeCache)

	metrics := fakeSys.GetMetrics()

	if metrics.CPUUsage != 50.0 {
		t.Fatalf("ожидали CPUUsage из кэша = %v, получили %v", 50.0, metrics.CPUUsage)
	}
}

func TestSystem_GetMetrics_CacheMiss(t *testing.T) {
	fakeCache := &FakeMetricsCache{getErr: ErrCacheMiss}

	fakeSys := newTestSystem(nil, fakeCache)
	fakeSys.metrics = Metrics{CPUUsage: 30}

	metrics := fakeSys.GetMetrics()

	if metrics.CPUUsage != 30.0 {
		t.Fatalf("ожидали CPUUsage = %v, получили %v", 30.0, metrics.CPUUsage)
	}
}

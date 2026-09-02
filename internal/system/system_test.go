package system

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type FakeRepository struct {
	metrics []Metrics
	alerts  []Alert

	saveMetricsErr    error
	saveAlertErr      error
	getActiveAlertErr error
	resolveAlertErr   error
	getMetricsErr     error

	nextAlertID int64
}

func (f *FakeRepository) SaveMetrics(metrics Metrics) error {
	if f.saveMetricsErr != nil {
		return f.saveMetricsErr
	}

	f.metrics = append(f.metrics, metrics)
	return nil
}

func (f *FakeRepository) SaveAlert(alert Alert) (int64, error) {
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

func (f *FakeRepository) GetAlerts(activeOnly bool) ([]Alert, error) {
	return nil, nil
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

func (f *FakeRepository) GetActiveAlert(alertType AlertType) (*Alert, error) {
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

func newTestSystem(fakeRepo *FakeRepository) *System {
	return NewSystem(fakeRepo)
}

func TestSystem_CollectMetrics(t *testing.T) {
	fakeErr := fmt.Errorf("database unavailable")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepository{
				metrics:        make([]Metrics, 0),
				saveMetricsErr: tt.repoErr,
			}

			system := newTestSystem(fakeRepo)

			err := system.CollectMetrics()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
			}
		})
	}
}

func TestSystem_ProcessAlerts_CreateAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]Alert, 0)}
	system := newTestSystem(fakeRepo)

	system.AlertCPU.consecutiveHigh = AlertTriggerCount
	metrics := Metrics{
		CPUUsage: 95.5,
	}

	err := system.processAlerts(metrics)
	if err != nil {
		t.Fatalf("ошибка обработки алерта: %v", err)
	}

	if len(fakeRepo.alerts) != 1 {
		t.Fatalf("ожидался 1 созданный алерт, получили %v", len(fakeRepo.alerts))
	}

	alert := fakeRepo.alerts[0]

	if alert.Type != AlertTypeHighCPU {
		t.Fatalf("ожидали тип алерта %v, получили %v", AlertTypeHighCPU, alert.Type)
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
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]Alert, 0)}

	// сохраняем алерт до теста, чтобы при добавлении нового алерта уже был активный в памяти
	_, err := fakeRepo.SaveAlert(Alert{
		Type: AlertTypeHighCPU,
	})
	if err != nil {
		t.Fatalf("не удалось сохранить алерт: %v", err)
	}
	system := newTestSystem(fakeRepo)

	system.AlertCPU.consecutiveHigh = AlertTriggerCount
	metrics := Metrics{
		CPUUsage: 95.5,
	}

	err = system.processAlerts(metrics)
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
		alerts:            make([]Alert, 0),
		getActiveAlertErr: fakeErr,
	}
	system := newTestSystem(fakeRepo)

	system.AlertCPU.consecutiveHigh = AlertTriggerCount
	metrics := Metrics{
		CPUUsage: 95.5,
	}

	err := system.processAlerts(metrics)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_CreateAlert_SaveAlertError(t *testing.T) {
	fakeErr := errors.New("не удалось сохранить алерт")

	fakeRepo := &FakeRepository{
		metrics:      make([]Metrics, 0),
		alerts:       make([]Alert, 0),
		saveAlertErr: fakeErr,
	}
	system := newTestSystem(fakeRepo)

	system.AlertCPU.consecutiveHigh = AlertTriggerCount
	metrics := Metrics{
		CPUUsage: 95.5,
	}

	err := system.processAlerts(metrics)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]Alert, 0)}

	alert := Alert{
		ID:         1,
		Type:       AlertTypeHighCPU,
		Resolved:   false,
		ResolvedAt: nil,
	}

	fakeRepo.alerts = append(fakeRepo.alerts, alert)

	system := newTestSystem(fakeRepo)
	system.AlertCPU.consecutiveNormal = AlertResolveCount

	err := system.processAlerts(Metrics{})
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
		alerts:            make([]Alert, 0),
		getActiveAlertErr: fakeErr,
	}

	system := newTestSystem(fakeRepo)
	system.AlertCPU.consecutiveNormal = AlertResolveCount

	err := system.processAlerts(Metrics{})
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert_ResolveError(t *testing.T) {
	fakeErr := errors.New("не удалось зарезолвить алерт")

	fakeRepo := &FakeRepository{
		metrics:         make([]Metrics, 0),
		alerts:          make([]Alert, 0),
		resolveAlertErr: fakeErr,
	}

	alert := Alert{
		ID:         1,
		Type:       AlertTypeHighCPU,
		Resolved:   false,
		ResolvedAt: nil,
	}

	fakeRepo.alerts = append(fakeRepo.alerts, alert)

	system := newTestSystem(fakeRepo)
	system.AlertCPU.consecutiveNormal = AlertResolveCount

	err := system.processAlerts(Metrics{})
	if !errors.Is(err, fakeErr) {
		t.Fatalf("ожидали ошибку %v, получили %v", fakeErr, err)
	}
}

func TestSystem_ProcessAlerts_ResolveAlert_NoActiveAlert(t *testing.T) {
	fakeRepo := &FakeRepository{metrics: make([]Metrics, 0), alerts: make([]Alert, 0)}

	system := newTestSystem(fakeRepo)
	system.AlertCPU.consecutiveNormal = AlertResolveCount

	err := system.processAlerts(Metrics{})
	if err != nil {
		t.Fatalf("не удалось обработать алерт: %v", err)
	}
}

func TestAlertState_Record(t *testing.T) {
	tests := []struct {
		name                        string
		values                      []float64
		threshold                   float64
		wantConsecutiveHigh         int
		wantConsecutiveNormal       int
		wantHighThresholdReached    bool
		wantResolveThresholdReached bool
	}{
		{
			name:                        "три HIGH измерения подряд",
			values:                      []float64{99.9, 99.9, 99.9},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         3,
			wantConsecutiveNormal:       0,
			wantHighThresholdReached:    true,
			wantResolveThresholdReached: false,
		},
		{
			name:                        "три NORMAL измерения подряд",
			values:                      []float64{19.9, 19.9, 19.9},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         0,
			wantConsecutiveNormal:       3,
			wantHighThresholdReached:    false,
			wantResolveThresholdReached: true,
		},
		{
			name:                        "два HIGH и одно NORMAL измерение",
			values:                      []float64{99.9, 99.9, 19.9},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         0,
			wantConsecutiveNormal:       1,
			wantHighThresholdReached:    false,
			wantResolveThresholdReached: false,
		},
		{
			name:                        "два NORMAL и одно HIGH измерение",
			values:                      []float64{19.9, 19.9, 99.9},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         1,
			wantConsecutiveNormal:       0,
			wantHighThresholdReached:    false,
			wantResolveThresholdReached: false,
		},
		{
			name:                        "три HIGH и один NORMAL",
			values:                      []float64{99.9, 99.9, 99.9, 19.9},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         0,
			wantConsecutiveNormal:       1,
			wantHighThresholdReached:    false,
			wantResolveThresholdReached: false,
		},
		{
			name:                        "измерения равные порогу",
			values:                      []float64{HighCPUThreshold, HighCPUThreshold, HighCPUThreshold},
			threshold:                   HighCPUThreshold,
			wantConsecutiveHigh:         0,
			wantConsecutiveNormal:       3,
			wantHighThresholdReached:    false,
			wantResolveThresholdReached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertState := AlertState{}

			for _, value := range tt.values {
				alertState.Record(value, tt.threshold)
			}

			if tt.wantConsecutiveHigh != alertState.consecutiveHigh {
				t.Fatalf("ожидали %v подряд HIGH измерений, получили %v",
					tt.wantConsecutiveHigh, alertState.consecutiveHigh)
			}

			if tt.wantConsecutiveNormal != alertState.consecutiveNormal {
				t.Fatalf("ожидали %v подряд NORMAL измерений, получили %v",
					tt.wantConsecutiveNormal, alertState.consecutiveNormal)
			}

			if tt.wantHighThresholdReached != alertState.HighThresholdReached() {
				t.Fatalf("ожидали %v от HighThresholdReached, получили %v",
					tt.wantHighThresholdReached, alertState.HighThresholdReached())
			}

			if tt.wantResolveThresholdReached != alertState.ResolveThresholdReached() {
				t.Fatalf("ожидали %v от ResolveThresholdReached, получили %v",
					tt.wantResolveThresholdReached, alertState.ResolveThresholdReached())
			}
		})
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

	system := newTestSystem(&fakeRepo)
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

	system := newTestSystem(&fakeRepo)
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

	system := newTestSystem(&fakeRepo)
	metrics, err := system.GetHistory(now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("не удалось получить историю измерений метрик: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("ожидалось 2 метрики, получили %v", len(metrics))
	}
}

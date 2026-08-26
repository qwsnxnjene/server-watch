package system

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type FakeRepository struct {
	metrics []Metrics
	err     error
}

func (f *FakeRepository) SaveMetrics(metrics Metrics) error {
	if f.err != nil {
		return f.err
	}

	f.metrics = append(f.metrics, metrics)
	return nil
}

func (f *FakeRepository) SaveAlert(alert Alert) (int64, error) {
	return 0, nil
}

func (f *FakeRepository) GetMetrics(from time.Time, to time.Time) ([]Metrics, error) {
	return nil, nil
}

func (f *FakeRepository) GetAlerts(activeOnly bool) ([]Alert, error) {
	return nil, nil
}

func (f *FakeRepository) ResolveAlert(id int64, resolvedAt time.Time) error {
	return nil
}

func (f *FakeRepository) GetActiveAlert(alertType AlertType) (*Alert, error) {
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
				metrics: make([]Metrics, 0),
				err:     tt.repoErr,
			}

			system := newTestSystem(fakeRepo)

			err := system.CollectMetrics()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
			}
		})
	}

	fakeRepo := &FakeRepository{
		metrics: make([]Metrics, 0),
		err:     nil,
	}

	system := newTestSystem(fakeRepo)

	err := system.CollectMetrics()
	if err != nil {
		t.Fatalf("не удалось собрать метрики: %v", err)
	}

	fakeRepoWithErr := &FakeRepository{
		metrics: make([]Metrics, 0),
		err:     fakeErr,
	}

	system = newTestSystem(fakeRepoWithErr)
	err = system.CollectMetrics()
	if !errors.Is(err, fakeErr) {
		t.Fatal("ошибка в SaveMetrics не повлияла на CollectMetrics")
	}
}

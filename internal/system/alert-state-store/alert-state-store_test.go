package alert_state_store

import (
	"server-watch/internal/system/model"
	"testing"
)

func TestInMemoryAlertStateStore_IncrementCount(t *testing.T) {
	store := NewInMemoryAlertStateStore()

	val, err := store.IncrementCount(model.AlertTypeHighCPU, model.ConditionHigh)
	if err != nil {
		t.Fatal(err)
	}

	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}

	val, err = store.IncrementCount(model.AlertTypeHighCPU, model.ConditionHigh)
	if err != nil {
		t.Fatal(err)
	}

	if val != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", val)
	}

	val, err = store.IncrementCount(model.AlertTypeHighCPU, model.ConditionNormal)
	if err != nil {
		t.Fatal(err)
	}

	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}

	val, err = store.IncrementCount(model.AlertTypeHighCPU, model.ConditionNormal)
	if err != nil {
		t.Fatal(err)
	}

	if val != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", val)
	}

	val, err = store.IncrementCount(model.AlertTypeHighCPU, model.ConditionHigh)
	if err != nil {
		t.Fatal(err)
	}

	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}
}

func TestInMemoryAlertStateStore_IncrementCount_InvalidType(t *testing.T) {
	store := NewInMemoryAlertStateStore()

	_, err := store.IncrementCount(model.AlertType("UNKNOWN"), model.ConditionHigh)
	if err == nil {
		t.Fatal("ожидали ошибку типа алерта, получили nil")
	}
}

func TestInMemoryAlertStateStore_SetActive(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "true", want: true},
		{name: "false", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryAlertStateStore()

			err := store.SetActive(model.AlertTypeHighCPU, tt.want)
			if err != nil {
				t.Fatal(err)
			}

			got, err := store.IsActive(model.AlertTypeHighCPU)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("ожидали статус алерта = %v, получили %v", tt.want, got)
			}
		})
	}
}

func TestInMemoryAlertStateStore_SetActive_ValueChanging(t *testing.T) {
	store := NewInMemoryAlertStateStore()

	val, err := store.IncrementCount(model.AlertTypeHighCPU, model.ConditionHigh)
	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}

	err = store.SetActive(model.AlertTypeHighCPU, true)
	if err != nil {
		t.Fatal(err)
	}

	val, err = store.IncrementCount(model.AlertTypeHighCPU, model.ConditionHigh)
	if err != nil {
		t.Fatal(err)
	}

	if val != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", val)
	}
}

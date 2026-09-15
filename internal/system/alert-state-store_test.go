package system

import (
	"testing"
)

func TestInMemoryAlertStateStore_IncrementCount(t *testing.T) {
	store := NewInMemoryAlertStateStore()

	val, err := store.IncrementCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatal(err)
	}

	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}

	val, err = store.IncrementCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatal(err)
	}

	if val != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", val)
	}

	err = store.ResetCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatalf("не удалось сбросить значение счетчика: %v", err)
	}

	val, err = store.IncrementCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatal(err)
	}

	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}
}

func TestInMemoryAlertStateStore_IncrementCount_InvalidType(t *testing.T) {
	store := NewInMemoryAlertStateStore()

	_, err := store.IncrementCount(AlertType("UNKNOWN"))
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

			err := store.SetActive(AlertTypeHighCPU, tt.want)
			if err != nil {
				t.Fatal(err)
			}

			got, err := store.IsActive(AlertTypeHighCPU)
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

	val, err := store.IncrementCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Fatalf("ожидали значение счетчика = 1, получили %v", val)
	}

	err = store.SetActive(AlertTypeHighCPU, true)
	if err != nil {
		t.Fatal(err)
	}

	val, err = store.IncrementCount(AlertTypeHighCPU)
	if err != nil {
		t.Fatal(err)
	}

	if val != 2 {
		t.Fatalf("ожидали значение счетчика = 2, получили %v", val)
	}
}

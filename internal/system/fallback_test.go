package system

import (
	"errors"
	"testing"
)

type mockAlertStateStore struct {
	incrementValue int64
	incrementErr   error
	incrementCalls int

	resetErr   error
	resetCalls int

	isActiveValue bool
	isActiveErr   error
	isActiveCalls int

	setActiveErr   error
	setActiveCalls int
	active         bool
}

func (m *mockAlertStateStore) IncrementCount(alertType AlertType, condition AlertCondition) (int64, error) {
	m.incrementCalls++
	return m.incrementValue, m.incrementErr
}

func (m *mockAlertStateStore) IsActive(alertType AlertType) (bool, error) {
	m.isActiveCalls++
	return m.isActiveValue, m.isActiveErr
}

func (m *mockAlertStateStore) SetActive(alertType AlertType, active bool) error {
	m.active = active
	m.setActiveCalls++
	return m.setActiveErr
}

func (m *mockAlertStateStore) GetState(alertType AlertType) (AlertState, error) {
	return AlertState{}, nil
}

func (m *mockAlertStateStore) SetState(alertType AlertType, state AlertState) error {
	return nil
}

func TestFallbackAlertStateStore_IncrementCount(t *testing.T) {
	tests := []struct {
		name string

		redisValue int64
		redisErr   error
		redisCalls int

		memValue int64
		memErr   error
		memCalls int

		wantValue int64
		wantErr   bool
	}{
		{
			name:       "redis доступен, память доступна",
			redisValue: 5,
			redisErr:   nil,
			redisCalls: 1,
			memValue:   10,
			memErr:     nil,
			memCalls:   0,
			wantValue:  5,
			wantErr:    false,
		},
		{
			name:       "redis недоступен, память доступна",
			redisValue: 5,
			redisErr:   ErrCacheMiss,
			redisCalls: 1,
			memValue:   10,
			memErr:     nil,
			memCalls:   1,
			wantValue:  10,
			wantErr:    false,
		},
		{
			name:       "redis доступен, память недоступна",
			redisValue: 5,
			redisErr:   nil,
			redisCalls: 1,
			memValue:   10,
			memErr:     errors.New("ошибка памяти"),
			memCalls:   0,
			wantValue:  5,
			wantErr:    false,
		},
		{
			name:       "redis недоступен, память недоступна",
			redisValue: 5,
			redisErr:   ErrCacheMiss,
			redisCalls: 1,
			memValue:   10,
			memErr:     errors.New("ошибка памяти"),
			memCalls:   1,
			wantValue:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisMock := &mockAlertStateStore{
				incrementValue: tt.redisValue,
				incrementErr:   tt.redisErr,
			}

			memMock := &mockAlertStateStore{
				incrementValue: tt.memValue,
				incrementErr:   tt.memErr,
			}

			store := NewFallbackAlertStateStore(memMock, redisMock)

			got, err := store.IncrementCount(AlertTypeHighCPU, ConditionHigh)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ожидали ошибку (или её отсутствие), получили %v", err)
			}

			if tt.wantValue != got {
				t.Fatalf("ожидали значение счетчика = %v, получили %v", tt.wantValue, got)
			}

			if tt.redisCalls != redisMock.incrementCalls {
				t.Fatalf("ожидали %v вызовов Redis, получили %v",
					tt.redisCalls, redisMock.incrementCalls)
			}

			if tt.memCalls != memMock.incrementCalls {
				t.Fatalf("ожидали %v вызовов in-memory, получили %v",
					tt.memCalls, memMock.incrementCalls)
			}
		})
	}
}

func TestFallbackAlertStateStore_IsActive(t *testing.T) {
	tests := []struct {
		name string

		redisErr   error
		redisValue bool
		redisCalls int

		memErr   error
		memValue bool
		memCalls int

		wantErr   bool
		wantValue bool
	}{
		{
			name:       "redis доступен, память доступна",
			redisErr:   nil,
			redisValue: true,
			redisCalls: 1,
			memErr:     nil,
			memValue:   false,
			memCalls:   0,
			wantValue:  true,
		},
		{
			name:       "redis недоступен, память доступна",
			redisErr:   ErrCacheMiss,
			redisValue: true,
			redisCalls: 1,
			memErr:     nil,
			memValue:   false,
			memCalls:   1,
			wantValue:  false,
		},
		{
			name:       "redis доступен, память недоступна",
			redisErr:   nil,
			redisValue: true,
			redisCalls: 1,
			memErr:     errors.New("ошибка памяти"),
			memValue:   false,
			memCalls:   0,
			wantValue:  true,
		},
		{
			name:       "redis недоступен, память недоступна",
			redisErr:   ErrCacheMiss,
			redisValue: true,
			redisCalls: 1,
			memErr:     errors.New("ошибка памяти"),
			memValue:   false,
			memCalls:   1,
			wantValue:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisMock := &mockAlertStateStore{
				isActiveErr:   tt.redisErr,
				isActiveValue: tt.redisValue,
			}

			memMock := &mockAlertStateStore{
				isActiveErr:   tt.memErr,
				isActiveValue: tt.memValue,
			}

			store := NewFallbackAlertStateStore(memMock, redisMock)

			got, err := store.IsActive(AlertTypeHighCPU)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ожидали ошибку (или её отсутствие), получили %v", err)
			}

			if got != tt.wantValue {
				t.Fatalf("ожидали значение = %v, получили %v", tt.wantValue, got)
			}

			if tt.redisCalls != redisMock.isActiveCalls {
				t.Fatalf("ожидали вызовов Redis = %v, получили %v",
					tt.redisCalls, redisMock.isActiveCalls)
			}

			if tt.memCalls != memMock.isActiveCalls {
				t.Fatalf("ожидали вызовов in-memory = %v, получили %v",
					tt.memCalls, memMock.isActiveCalls)
			}
		})
	}
}

func TestFallbackAlertStateStore_SetActive(t *testing.T) {
	tests := []struct {
		name string

		active bool

		redisErr   error
		redisCalls int

		memErr   error
		memCalls int

		wantRedisActive bool
		wantMemActive   bool
		wantErr         bool
	}{
		{
			name:            "redis доступен, Active true",
			active:          true,
			redisErr:        nil,
			redisCalls:      1,
			memCalls:        0,
			wantRedisActive: true,
			wantMemActive:   false,
			wantErr:         false,
		},
		{
			name:            "redis доступен, Active false",
			active:          false,
			redisErr:        nil,
			redisCalls:      1,
			memCalls:        0,
			wantRedisActive: false,
			wantMemActive:   false,
			wantErr:         false,
		},
		{
			name:            "redis недоступен, память доступна",
			active:          true,
			redisErr:        ErrCacheMiss,
			redisCalls:      1,
			memErr:          nil,
			memCalls:        1,
			wantRedisActive: true,
			wantMemActive:   true,
			wantErr:         false,
		},
		{
			name:            "redis недоступен, память доступна, Active false",
			active:          false,
			redisErr:        ErrCacheMiss,
			redisCalls:      1,
			memErr:          nil,
			memCalls:        1,
			wantRedisActive: false,
			wantMemActive:   false,
			wantErr:         false,
		},
		{
			name:            "redis недоступен, память недоступна",
			active:          true,
			redisErr:        ErrCacheMiss,
			redisCalls:      1,
			memErr:          errors.New("ошибка памяти"),
			memCalls:        1,
			wantRedisActive: true,
			wantMemActive:   true,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisMock := &mockAlertStateStore{
				setActiveErr: tt.redisErr,
			}

			memMock := &mockAlertStateStore{
				setActiveErr: tt.memErr,
			}

			store := NewFallbackAlertStateStore(memMock, redisMock)

			err := store.SetActive(AlertTypeHighCPU, tt.active)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ожидали ошибку = %v, получили %v",
					tt.wantErr,
					err,
				)
			}

			if tt.redisCalls != redisMock.setActiveCalls {
				t.Fatalf(
					"ожидали вызовов Redis = %v, получили %v",
					tt.redisCalls,
					redisMock.setActiveCalls,
				)
			}

			if tt.memCalls != memMock.setActiveCalls {
				t.Fatalf(
					"ожидали вызовов in-memory = %v, получили %v",
					tt.memCalls,
					memMock.setActiveCalls,
				)
			}

			if redisMock.active != tt.wantRedisActive {
				t.Fatalf(
					"ожидали Active в Redis = %v, получили %v",
					tt.wantRedisActive,
					redisMock.active,
				)
			}

			if memMock.active != tt.wantMemActive {
				t.Fatalf(
					"ожидали Active в in-memory = %v, получили %v",
					tt.wantMemActive,
					memMock.active,
				)
			}
		})
	}
}

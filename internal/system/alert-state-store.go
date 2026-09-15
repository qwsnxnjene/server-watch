package system

import (
	"errors"
	"sync"
)

type alertState struct {
	count  int64
	active bool
}

type InMemoryAlertStateStore struct {
	mu     sync.RWMutex
	states map[AlertType]alertState
}

func NewInMemoryAlertStateStore() *InMemoryAlertStateStore {
	return &InMemoryAlertStateStore{
		states: make(map[AlertType]alertState),
	}
}

func validAlertType(alertType AlertType) bool {
	return alertType == AlertTypeHighMem || alertType == AlertTypeHighCPU
}

func (i *InMemoryAlertStateStore) IncrementCount(alertType AlertType) (int64, error) {
	var value int64

	if !validAlertType(alertType) {
		return 0, errors.New("некорректный тип алерта")
	}

	i.mu.Lock()
	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = alertState{count: 1}
		value = 1
	} else {
		state.count++
		value = state.count
		i.states[alertType] = state
	}
	i.mu.Unlock()

	return value, nil
}

func (i *InMemoryAlertStateStore) ResetCount(alertType AlertType) error {
	if !validAlertType(alertType) {
		return errors.New("некорректный тип алерта")
	}

	i.mu.Lock()
	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = alertState{count: 0}
	} else {
		state.count = 0
		i.states[alertType] = state
	}
	i.mu.Unlock()

	return nil
}

func (i *InMemoryAlertStateStore) IsActive(alertType AlertType) (bool, error) {
	if !validAlertType(alertType) {
		return false, errors.New("некорректный тип алерта")
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	if state, ok := i.states[alertType]; !ok {
		return false, nil
	} else {
		return state.active, nil
	}
}

func (i *InMemoryAlertStateStore) SetActive(alertType AlertType, active bool) error {
	if !validAlertType(alertType) {
		return errors.New("некорректный тип алерта")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = alertState{
			active: active,
		}
	} else {
		state.active = active
		i.states[alertType] = state
	}

	return nil
}

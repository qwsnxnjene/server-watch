package system

import (
	"errors"
	"sync"
)

type AlertState struct {
	Count     int64
	Condition AlertCondition
	Active    bool
}

type InMemoryAlertStateStore struct {
	mu     sync.RWMutex
	states map[AlertType]AlertState
}

func NewInMemoryAlertStateStore() *InMemoryAlertStateStore {
	return &InMemoryAlertStateStore{
		states: make(map[AlertType]AlertState),
	}
}

func validAlertType(alertType AlertType) bool {
	return alertType == AlertTypeHighMem || alertType == AlertTypeHighCPU
}

func (i *InMemoryAlertStateStore) IncrementCount(alertType AlertType, condition AlertCondition) (int64, error) {
	if !validAlertType(alertType) {
		return 0, errors.New("некорректный тип алерта")
	}

	var value int64

	i.mu.Lock()
	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = AlertState{Count: 1, Condition: condition}
		value = 1
	} else {
		if state.Condition != condition {
			// состояния различны - ставим новое состояние и сбрасываем счетчик
			state.Condition = condition
			state.Count = 1
			value = 1
			i.states[alertType] = state
		} else {
			// состояния одинаковые - просто обновляем счетчик
			state.Count++
			value = state.Count
			i.states[alertType] = state
		}
	}
	i.mu.Unlock()

	return value, nil
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
		return state.Active, nil
	}
}

func (i *InMemoryAlertStateStore) SetActive(alertType AlertType, active bool) error {
	if !validAlertType(alertType) {
		return errors.New("некорректный тип алерта")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = AlertState{
			Active: active,
		}
	} else {
		state.Active = active
		i.states[alertType] = state
	}

	return nil
}

func (i *InMemoryAlertStateStore) GetState(alertType AlertType) (AlertState, error) {
	if !validAlertType(alertType) {
		return AlertState{}, errors.New("некорректный тип алерта")
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	if state, ok := i.states[alertType]; ok {
		return state, nil
	}

	return AlertState{}, nil
}

func (i *InMemoryAlertStateStore) SetState(alertType AlertType, state AlertState) error {
	return nil
}

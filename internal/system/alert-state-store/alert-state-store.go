package alert_state_store

import (
	"errors"
	"server-watch/internal/system/model"
	"sync"
)

// InMemoryAlertStateStore хранит состояния алертов в памяти.
// Доступ к состояниям защищён mutex для безопасной работы из нескольких горутин
type InMemoryAlertStateStore struct {
	mu     sync.RWMutex
	states map[model.AlertType]model.AlertState
}

// NewInMemoryAlertStateStore создаёт пустое in-memory хранилище состояний алертов
func NewInMemoryAlertStateStore() *InMemoryAlertStateStore {
	return &InMemoryAlertStateStore{
		states: make(map[model.AlertType]model.AlertState),
	}
}

func validAlertType(alertType model.AlertType) bool {
	return alertType == model.AlertTypeHighMem || alertType == model.AlertTypeHighCPU
}

// IncrementCount увеличивает счётчик для указанного типа и условия.
// При изменении условия счётчик начинается заново с единицы
func (i *InMemoryAlertStateStore) IncrementCount(alertType model.AlertType, condition model.AlertCondition) (int64, error) {
	if !validAlertType(alertType) {
		return 0, errors.New("некорректный тип алерта")
	}

	var value int64

	i.mu.Lock()
	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = model.AlertState{Count: 1, Condition: condition}
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

// IsActive возвращает текущий статус алерта указанного типа
func (i *InMemoryAlertStateStore) IsActive(alertType model.AlertType) (bool, error) {
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

// SetActive устанавливает статус алерта указанного типа
func (i *InMemoryAlertStateStore) SetActive(alertType model.AlertType, active bool) error {
	if !validAlertType(alertType) {
		return errors.New("некорректный тип алерта")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if state, ok := i.states[alertType]; !ok {
		i.states[alertType] = model.AlertState{
			Active: active,
		}
	} else {
		state.Active = active
		i.states[alertType] = state
	}

	return nil
}

// GetState возвращает полное состояние алерта указанного типа.
// Если состояние ещё не сохранено, возвращается его нулевое значение без ошибки
func (i *InMemoryAlertStateStore) GetState(alertType model.AlertType) (model.AlertState, error) {
	if !validAlertType(alertType) {
		return model.AlertState{}, errors.New("некорректный тип алерта")
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	if state, ok := i.states[alertType]; ok {
		return state, nil
	}

	return model.AlertState{}, nil
}

func (i *InMemoryAlertStateStore) SetState(alertType model.AlertType, state model.AlertState) error {
	return nil
}

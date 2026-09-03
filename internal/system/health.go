package system

import "time"

// GetHealth возвращает время последнего успешного чтения метрик и последнюю ошибку
func (s *System) GetHealth() (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSuccess, s.lastError
}

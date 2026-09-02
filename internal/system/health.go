package system

import "time"

func (s *System) GetHealth() (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSuccess, s.lastError
}

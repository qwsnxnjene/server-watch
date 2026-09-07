package system

import (
	"log/slog"
	"server-watch/internal/config"
)

type ConfigUpdate struct {
	CPUThreshold *float64 `json:"cpu_threshold,omitempty"`
	MemThreshold *float64 `json:"mem_threshold,omitempty"`
	TriggerCount *int     `json:"trigger_count,omitempty"`
	ResolveCount *int     `json:"resolve_count,omitempty"`
	SlackEnabled *bool    `json:"slack_enabled,omitempty"`
	SlackURL     *string  `json:"slack_url,omitempty"`
}

func (s *System) UpdateConfig(updatedCfg ConfigUpdate) error {
	s.mu.RLock()
	currCfg := s.config
	s.mu.RUnlock()

	if updatedCfg.CPUThreshold != nil {
		currCfg.CPUThreshold = *updatedCfg.CPUThreshold
	}
	if updatedCfg.MemThreshold != nil {
		currCfg.MemThreshold = *updatedCfg.MemThreshold
	}
	if updatedCfg.TriggerCount != nil {
		currCfg.TriggerCount = *updatedCfg.TriggerCount
	}
	if updatedCfg.ResolveCount != nil {
		currCfg.ResolveCount = *updatedCfg.ResolveCount
	}
	if updatedCfg.SlackEnabled != nil {
		currCfg.SlackEnabled = *updatedCfg.SlackEnabled
	}
	if updatedCfg.SlackURL != nil {
		currCfg.SlackURL = *updatedCfg.SlackURL
	}

	err := config.Validate(currCfg)
	if err != nil {
		return err
	}

	// по ТЗ обновляем конфиг в любом случае, даже если в файл сохранить не удалось
	s.mu.Lock()
	s.config = currCfg
	s.mu.Unlock()

	err = config.SaveToYAML(s.configPath, currCfg)
	if err != nil {
		slog.Error(
			"не удалось сохранить конфигурацию в файл, новый настройки применены только в памяти",
			"path", s.configPath,
			"error", err,
		)
		return err
	}

	return nil
}

func (s *System) GetConfig() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

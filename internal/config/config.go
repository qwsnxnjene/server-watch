package config

import (
	"errors"
	"fmt"
	"log/slog"
	url2 "net/url"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

var ErrInvalidYAML = errors.New("некорректный YAML")
var ErrInvalidConfig = errors.New("некорректная конфигурация")

type Config struct {
	CPUThreshold float64 `yaml:"cpu_threshold"`
	MemThreshold float64 `yaml:"mem_threshold"`
	TriggerCount int     `yaml:"trigger_count"`
	ResolveCount int     `yaml:"resolve_count"`
	SlackEnabled bool    `yaml:"slack_enabled"`
	SlackURL     string  `yaml:"slack_url"`
}

// интеграция Config в main и в System

func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	err := loadFromYAML(path, &cfg)
	if err != nil {
		if errors.Is(err, ErrInvalidYAML) {
			return Config{}, err
		}

		if errors.Is(err, os.ErrNotExist) {
			slog.Info("файл конфигурации не найден", "path", path)
			err = SaveToYAML(path, cfg)
			if err != nil {
				return Config{}, err
			}
		} else {
			return Config{}, fmt.Errorf("не удалось загрузить конфиг из файла %v: %w",
				path, err)
		}
	}

	err = loadFromEnv(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("не удалось прочитать переменную окружения: %w", err)
	}

	if err = Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("некорректная конфигурация: %w", err)
	}

	return cfg, nil
}

func Validate(cfg Config) error {
	if cfg.CPUThreshold < 0 || cfg.CPUThreshold > 100 {
		return fmt.Errorf("%w: значение CPUThreshold должно быть от 0 до 100, получили: %v",
			ErrInvalidConfig, cfg.CPUThreshold)
	}

	if cfg.MemThreshold < 0 || cfg.MemThreshold > 100 {
		return fmt.Errorf("%w: значение MemThreshold должно быть от 0 до 100, получили: %v",
			ErrInvalidConfig, cfg.MemThreshold)
	}

	if cfg.TriggerCount <= 0 {
		return fmt.Errorf("%w: значение TriggerCount должно быть >0, получили: %v",
			ErrInvalidConfig, cfg.TriggerCount)
	}

	if cfg.ResolveCount <= 0 {
		return fmt.Errorf("%w: значение ResolveCount должно быть >0, получили: %v",
			ErrInvalidConfig, cfg.ResolveCount)
	}

	skipValidation := cfg.SlackURL == "" && !cfg.SlackEnabled

	if !validateURL(cfg.SlackURL) && !skipValidation {
		return fmt.Errorf("%w: значение SlackURL некорректно: %v",
			ErrInvalidConfig, cfg.SlackURL)
	}

	return nil
}

func validateURL(value string) bool {
	url, err := url2.Parse(value)
	if err != nil {
		return false
	}

	scheme := url.Scheme
	host := url.Host

	if scheme != "http" && scheme != "https" {
		return false
	}

	if host == "" {
		return false
	}

	return true
}

func DefaultConfig() Config {
	return Config{
		CPUThreshold: 80,
		MemThreshold: 90,
		TriggerCount: 3,
		ResolveCount: 3,
		SlackEnabled: false,
	}
}

func loadFromEnv(cfg *Config) error {
	value, exists := os.LookupEnv("CPU_THRESHOLD")
	if exists {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("ошибка чтения CPU_THRESHOLD из окружения: %w", err)
		}

		cfg.CPUThreshold = val
	}

	value, exists = os.LookupEnv("MEM_THRESHOLD")
	if exists {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("ошибка чтения MEM_THRESHOLD из окружения: %w", err)
		}

		cfg.MemThreshold = val
	}

	value, exists = os.LookupEnv("TRIGGER_COUNT")
	if exists {
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("ошибка чтения TRIGGER_COUNT из окружения: %w", err)
		}

		cfg.TriggerCount = val
	}

	value, exists = os.LookupEnv("RESOLVE_COUNT")
	if exists {
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("ошибка чтения RESOLVE_COUNT из окружения: %w", err)
		}

		cfg.ResolveCount = val
	}

	value, exists = os.LookupEnv("SLACK_ENABLED")
	if exists {
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("ошибка чтения SLACK_ENABLED из окружения: %w", err)
		}

		cfg.SlackEnabled = val
	}

	value, exists = os.LookupEnv("SLACK_URL")
	if exists {
		cfg.SlackURL = value
	}

	return nil
}

func loadFromYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл конфигурации %v: %w", path, err)
	}

	if err = yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	return nil
}

func SaveToYAML(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("не удалось закодировать конфигурацию в yaml: %w", err)
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("не удалось записать конфигурацию: %w", err)
	}

	slog.Info("создан файл конфигурации", "path", path)
	return nil
}

package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_CPUThreshold(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{
			name:    "корректное значение",
			value:   67.6,
			wantErr: false,
		},
		{
			name:    "отрицательное значение",
			value:   -23.3,
			wantErr: true,
		},
		{
			name:    "значение >100",
			value:   101.2,
			wantErr: true,
		},
		{
			name:    "граничное значение 0",
			value:   0,
			wantErr: false,
		},
		{
			name:    "граничное значение 100",
			value:   100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.CPUThreshold = tt.value

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, ожидали %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MemThreshold(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{
			name:    "корректное значение",
			value:   67.6,
			wantErr: false,
		},
		{
			name:    "отрицательное значение",
			value:   -23.3,
			wantErr: true,
		},
		{
			name:    "значение >100",
			value:   101.2,
			wantErr: true,
		},
		{
			name:    "граничное значение 0",
			value:   0,
			wantErr: false,
		},
		{
			name:    "граничное значение 100",
			value:   100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.MemThreshold = tt.value

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, ожидали %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_TriggerCount(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{
			name:    "корректное значение",
			value:   3,
			wantErr: false,
		},
		{
			name:    "отрицательное значение",
			value:   -1,
			wantErr: true,
		},
		{
			name:    "значение =0",
			value:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.TriggerCount = tt.value

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, ожидали %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ResolveCount(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{
			name:    "корректное значение",
			value:   3,
			wantErr: false,
		},
		{
			name:    "отрицательное значение",
			value:   -1,
			wantErr: true,
		},
		{
			name:    "значение =0",
			value:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ResolveCount = tt.value

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, ожидали %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Slack(t *testing.T) {
	tests := []struct {
		name         string
		slackEnabled bool
		slackUrl     string
		wantErr      bool
	}{
		{
			name:         "Slack выключен + пустой URL",
			slackEnabled: false,
			slackUrl:     "",
			wantErr:      false,
		},
		{
			name:         "Slack выключен + валидный URL",
			slackEnabled: false,
			slackUrl:     "https://hooks.slack.com/services/xxx/yyy/zzz",
			wantErr:      false,
		},
		{
			name:         "Slack выключен + невалидный URL",
			slackEnabled: false,
			slackUrl:     "ftp://example.com",
			wantErr:      true,
		},
		{
			name:         "Slack включен + валидный URL",
			slackEnabled: true,
			slackUrl:     "https://hooks.slack.com/services/xxx/yyy/zzz",
			wantErr:      false,
		},
		{
			name:         "Slack включен + невалидный URL",
			slackEnabled: true,
			slackUrl:     "ftp://example.com",
			wantErr:      true,
		},
		{
			name:         "Slack включен + пустой URL",
			slackEnabled: true,
			slackUrl:     "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.SlackEnabled = tt.slackEnabled
			cfg.SlackURL = tt.slackUrl

			err := Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, ожидали %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_FromYAML(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Config
	}{
		{
			name: "полный конфиг",
			data: []byte(`
cpu_threshold: 70
mem_threshold: 85
trigger_count: 5
resolve_count: 4
slack_enabled: true
slack_url: "https://example.com/webhook"
`),
			want: Config{
				CPUThreshold: 70,
				MemThreshold: 85,
				TriggerCount: 5,
				ResolveCount: 4,
				SlackEnabled: true,
				SlackURL:     "https://example.com/webhook",
			},
		},
		{
			name: "неполный конфиг",
			data: []byte("cpu_threshold: 70"),
			want: Config{
				CPUThreshold: 70,
				MemThreshold: 90,
				TriggerCount: 3,
				ResolveCount: 3,
				SlackEnabled: false,
				SlackURL:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")

			err := os.WriteFile(path, tt.data, 0600)
			if err != nil {
				t.Fatalf("не удалось записать тестовый конфиг: %v", err)
			}

			got, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Load() = %+v, ожидали %+v", got, tt.want)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := DefaultConfig()

	if got != want {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("не удалось прочитать созданный файл: %v", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("не удалось декодировать yaml: %v", err)
	}

	if cfg != want {
		t.Fatalf("ожидали %+v, получили %+v", want, cfg)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	data := []byte("cpu_threshold: [")
	wantErr := ErrInvalidYAML

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatalf("не удалось записать тестовый конфиг: %v", err)
	}

	_, err = Load(path)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Load() error = %v, ожидали %v", err, wantErr)
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	data := []byte("cpu_threshold: 101")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatalf("не удалось записать тестовый конфиг: %v", err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("нет ошибки валидации при невалидных данных")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := []byte(`
cpu_threshold: 70
mem_threshold: 85
trigger_count: 5
resolve_count: 4
slack_enabled: true
slack_url: "https://example.com/webhook"
`)

	err := os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatalf("не удалось записать тестовый конфиг: %v", err)
	}

	env := map[string]string{
		"CPU_THRESHOLD": "70",
		"MEM_THRESHOLD": "85",
		"TRIGGER_COUNT": "5",
		"RESOLVE_COUNT": "4",
		"SLACK_ENABLED": "true",
		"SLACK_URL":     "https://example.com/webhook",
	}

	want := Config{
		CPUThreshold: 70,
		MemThreshold: 85,
		TriggerCount: 5,
		ResolveCount: 4,
		SlackEnabled: true,
		SlackURL:     "https://example.com/webhook",
	}

	for k, v := range env {
		err = os.Setenv(k, v)
		if err != nil {
			t.Fatalf("не удалось установить переменную окружения: %v", err)
		}
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got != want {
		t.Fatalf("Load() = %+v, ожидали %+v", got, want)
	}
}

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"server-watch/internal/system"
	"testing"
	"time"
)

type FakeSystem struct {
	metrics system.Metrics

	lastSuccess time.Time
	lastError   error

	historyErr  error
	history     []system.Metrics
	historyFrom time.Time
	historyTo   time.Time

	alertsErr        error
	alertsActiveOnly bool
	alerts           []system.Alert
}

func (f *FakeSystem) GetMetrics() system.Metrics {
	return f.metrics
}

func (f *FakeSystem) GetHistory(from, to time.Time) ([]system.Metrics, error) {
	f.historyFrom = from
	f.historyTo = to

	if f.historyErr != nil {
		return nil, f.historyErr
	}

	return f.history, nil
}

func (f *FakeSystem) GetAlerts(activeOnly bool) ([]system.Alert, error) {
	f.alertsActiveOnly = activeOnly

	if f.alertsErr != nil {
		return nil, f.alertsErr
	}

	return f.alerts, nil
}

func (f *FakeSystem) GetHealth() (time.Time, error) {
	return f.lastSuccess, f.lastError
}

func TestHandler_MetricsHandler(t *testing.T) {
	fakeSystem := &FakeSystem{
		metrics: system.Metrics{
			CPUUsage:   25.5,
			MemUsage:   40.0,
			MemUsedMB:  4000,
			MemTotalMB: 10000,
			DiskUsage:  50.0,
			DiskUsed:   50,
			DiskTotal:  100,
		},
	}

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewHandler(fakeSystem, prometheusHandler)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/json")
	rw := httptest.NewRecorder()

	handler.MetricsHandler(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("ожидался статус %v, получен %v", http.StatusOK, rw.Code)
	}

	var got MetricsResponse
	err := json.NewDecoder(rw.Body).Decode(&got)
	if err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}

	want := MetricsResponse{
		CPUPercent:  25.5,
		MemPercent:  40.0,
		MemUsedMB:   4000,
		MemTotalMB:  10000,
		DiskPercent: 50.0,
		DiskUsedGB:  50,
		DiskTotalGB: 100,
	}

	if got != want {
		t.Fatalf("получили неожиданные метрики:\nожидали: %+v\nполучили: %+v", want, got)
	}

	if got := rw.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("ожидался Content-Type application/json, получен %q", got)
	}
}

func TestAcceptsPrometheus(t *testing.T) {
	tests := []struct {
		name    string
		accept  string
		want    bool
		wantErr bool
	}{
		{
			name:   "Prometheus text",
			accept: "text/plain",
			want:   true,
		},
		{
			name:   "OpenMetrics",
			accept: "application/openmetrics-text",
			want:   true,
		},
		{
			name:   "JSON",
			accept: "application/json",
			want:   false,
		},
		{
			name:   "Accept отсутствует",
			accept: "",
			want:   false,
		},
		{
			name:   "Prometheus text disabled",
			accept: "text/plain;q=0",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			got, err := acceptsPrometheus(req)

			if (err != nil) != tt.wantErr {
				t.Fatalf("acceptsPrometheus() error = %v, ожидали = %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("acceptsPrometheus() = %v, ожидали %v", got, tt.want)
			}
		})
	}
}

func TestHandler_HealthHandler(t *testing.T) {
	fakeErr := errors.New("ошибка сбора метрик")

	tests := []struct {
		name             string
		lastSuccess      time.Time
		lastError        error
		wantStatusCode   int
		wantHealthStatus string
	}{
		{
			name:             "еще нет успешных сборов метрик",
			lastSuccess:      time.Time{},
			lastError:        nil,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantHealthStatus: "unhealthy",
		},
		{
			name:             "последний сбор завершился ошибкой",
			lastSuccess:      time.Now(),
			lastError:        fakeErr,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantHealthStatus: "unhealthy",
		},
		{
			name:             "метрики устарели",
			lastSuccess:      time.Now().Add(-time.Second * 11),
			lastError:        nil,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantHealthStatus: "unhealthy",
		},
		{
			name:             "система работает нормально",
			lastSuccess:      time.Now(),
			lastError:        nil,
			wantStatusCode:   http.StatusOK,
			wantHealthStatus: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSystem := &FakeSystem{
				lastError:   tt.lastError,
				lastSuccess: tt.lastSuccess,
			}

			prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := NewHandler(fakeSystem, prometheusHandler)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rw := httptest.NewRecorder()

			handler.HealthHandler(rw, req)

			if rw.Code != tt.wantStatusCode {
				t.Fatalf("ожидался код ответа %v, получили %v", tt.wantStatusCode, rw.Code)
			}

			var got HealthResponse
			err := json.NewDecoder(rw.Body).Decode(&got)
			if err != nil {
				t.Fatalf("не удалось декодировать ответ: %v", err)
			}

			if got.Status != tt.wantHealthStatus {
				t.Fatalf("ожидался статус %v, получен %v", tt.wantHealthStatus, got.Status)
			}

			if got.Status == "ok" {
				if got.LastCollection == "" {
					t.Fatal("не указано время последнего сбора данных")
				}
			}
		})
	}
}

func TestHandler_HistoryHandler_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	from := now.Add(-10 * time.Minute)
	to := now

	fakeSystem := &FakeSystem{
		history: []system.Metrics{
			{
				CPUUsage: 90.3,
			},
			{
				CPUUsage: 72.5,
			},
		},
	}

	query := "?from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339)

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewHandler(fakeSystem, prometheusHandler)

	req := httptest.NewRequest(http.MethodGet, "/history"+query, nil)
	rw := httptest.NewRecorder()

	handler.HistoryHandler(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("ожидался код ответа %v, получили %v", http.StatusOK, rw.Code)
	}

	var got HistoryResponse
	err := json.NewDecoder(rw.Body).Decode(&got)
	if err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}

	want := HistoryResponse{Metrics: []MetricsResponse{
		{
			CPUPercent: 90.3,
		},
		{
			CPUPercent: 72.5,
		},
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatal("ожидаемый ответ не совпадает с полученным")
	}
}

func TestHandler_HistoryHandler_QueryParams(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	from := now.Add(-10 * time.Minute)
	to := now

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
		wantFrom       time.Time
		wantTo         time.Time
	}{
		{
			name:           "оба параметра переданы и корректны",
			query:          "?from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339),
			wantStatusCode: http.StatusOK,
			wantFrom:       from,
			wantTo:         to,
		},
		{
			name:           "только параметр from передан и корректен",
			query:          "?from=" + from.Format(time.RFC3339),
			wantStatusCode: http.StatusOK,
			wantFrom:       from,
			wantTo:         time.Time{},
		},
		{
			name:           "только параметр to передан и корректен",
			query:          "?to=" + to.Format(time.RFC3339),
			wantStatusCode: http.StatusOK,
			wantFrom:       time.Time{},
			wantTo:         to,
		},
		{
			name:           "параметры не переданы",
			query:          "",
			wantStatusCode: http.StatusOK,
			wantFrom:       time.Time{},
			wantTo:         time.Time{},
		},
		{
			name:           "параметр from не валиден",
			query:          "?from=invalid&to=" + to.Format(time.RFC3339),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "параметр to не валиден",
			query:          "?from=" + to.Format(time.RFC3339) + "&to=invalid",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "параметр from идет после to",
			query:          "?from=" + to.Format(time.RFC3339) + "&to=" + from.Format(time.RFC3339),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSystem := &FakeSystem{}

			prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := NewHandler(fakeSystem, prometheusHandler)

			req := httptest.NewRequest(http.MethodGet, "/history"+tt.query, nil)
			rw := httptest.NewRecorder()

			before := time.Now().UTC().Truncate(time.Second)
			handler.HistoryHandler(rw, req)
			after := time.Now().UTC().Truncate(time.Second)

			if rw.Code != tt.wantStatusCode {
				t.Fatalf("ожидался код ответа %v, получили %v", tt.wantStatusCode, rw.Code)
			}

			if tt.wantStatusCode == http.StatusOK {
				if tt.wantFrom.IsZero() {
					wantFromMin := before.Add(-time.Hour)
					wantFromMax := after.Add(-time.Hour)

					if fakeSystem.historyFrom.Before(wantFromMin) || fakeSystem.historyFrom.After(wantFromMax) {
						t.Fatalf("значение from ожидалось в диапазоне от %v до %v, получили %v",
							wantFromMin, wantFromMax, fakeSystem.historyFrom)
					}
				} else {
					if !fakeSystem.historyFrom.Equal(tt.wantFrom) {
						t.Fatalf("значения параметра from не совпадают\nожидали: %v, получили: %v",
							tt.wantFrom, fakeSystem.historyFrom)
					}
				}

				if tt.wantTo.IsZero() {
					wantFromMin := before
					wantFromMax := after

					if fakeSystem.historyTo.Before(wantFromMin) || fakeSystem.historyTo.After(wantFromMax) {
						t.Fatalf("значение to ожидалось в диапазоне от %v до %v, получили %v",
							wantFromMin, wantFromMax, fakeSystem.historyTo)
					}
				} else {
					if !fakeSystem.historyTo.Equal(tt.wantTo) {
						t.Fatalf("значения параметра to не совпадают\nожидали: %v, получили: %v",
							tt.wantTo, fakeSystem.historyTo)
					}
				}
			}
		})
	}
}

func TestHandler_HistoryHandler_GetHistoryError(t *testing.T) {
	fakeErr := errors.New("не удалось получить историю измерения метрик")

	now := time.Now().UTC().Truncate(time.Second)
	from := now.Add(-10 * time.Minute)
	to := now

	fakeSystem := &FakeSystem{
		historyErr: fakeErr,
	}

	query := "?from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339)

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewHandler(fakeSystem, prometheusHandler)

	req := httptest.NewRequest(http.MethodGet, "/history"+query, nil)
	rw := httptest.NewRecorder()

	handler.HistoryHandler(rw, req)

	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("ожидался код ответа %v, получили %v", http.StatusInternalServerError, rw.Code)
	}
}

func TestHandler_AlertsHandler_QueryParam(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantStatusCode int
		wantActiveOnly bool
	}{
		{
			name:           "параметр не указан",
			query:          "",
			wantStatusCode: http.StatusOK,
			wantActiveOnly: false,
		},
		{
			name:           "параметр = true",
			query:          "?active_only=true",
			wantStatusCode: http.StatusOK,
			wantActiveOnly: true,
		},
		{
			name:           "параметр = false",
			query:          "?active_only=false",
			wantStatusCode: http.StatusOK,
			wantActiveOnly: false,
		},
		{
			name:           "параметр указан некорректно",
			query:          "?active_only=invalid",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSystem := &FakeSystem{}

			prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := NewHandler(fakeSystem, prometheusHandler)

			req := httptest.NewRequest(http.MethodGet, "/alerts"+tt.query, nil)
			rw := httptest.NewRecorder()

			handler.AlertsHandler(rw, req)

			if rw.Code != tt.wantStatusCode {
				t.Fatalf("ожидался код ответа %v, получили %v",
					tt.wantStatusCode, rw.Code)
			}

			if tt.wantStatusCode != http.StatusOK {
				return
			}

			if fakeSystem.alertsActiveOnly != tt.wantActiveOnly {
				t.Fatalf("ожидали activeOnly = %v, получили %v",
					tt.wantActiveOnly, fakeSystem.alertsActiveOnly)
			}
		})
	}
}

func TestHandler_AlertsHandler_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	alerts := []system.Alert{
		{
			ID:         1,
			Type:       system.AlertTypeHighCPU,
			Timestamp:  time.Time{},
			Threshold:  system.HighCPUThreshold,
			Resolved:   false,
			ResolvedAt: nil,
			Value:      88.9,
		},
		{
			ID:         2,
			Type:       system.AlertTypeHighMem,
			Timestamp:  time.Time{},
			Threshold:  system.HighMemThreshold,
			Resolved:   true,
			ResolvedAt: &now,
			Value:      95.4,
		},
	}

	wantResponse := AlertsResponse{
		Alerts: []AlertResponse{
			{
				ID:         1,
				Type:       system.AlertTypeHighCPU,
				Timestamp:  time.Time{},
				Threshold:  system.HighCPUThreshold,
				Resolved:   false,
				ResolvedAt: nil,
				Value:      88.9,
			},
			{
				ID:         2,
				Type:       system.AlertTypeHighMem,
				Timestamp:  time.Time{},
				Threshold:  system.HighMemThreshold,
				Resolved:   true,
				ResolvedAt: &now,
				Value:      95.4,
			},
		},
	}

	fakeSystem := &FakeSystem{
		alerts: alerts,
	}

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewHandler(fakeSystem, prometheusHandler)

	req := httptest.NewRequest(http.MethodGet, "/alerts?active_only=true", nil)
	rw := httptest.NewRecorder()

	handler.AlertsHandler(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("ожидался код ответа %v, получили %v",
			http.StatusOK, rw.Code)
	}

	var got AlertsResponse
	err := json.NewDecoder(rw.Body).Decode(&got)
	if err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}

	if !reflect.DeepEqual(got, wantResponse) {
		t.Fatal("ожидаемый ответ не совпадает с полученным")
	}
}

func TestHandler_AlertsHandler_GetAlertsError(t *testing.T) {
	fakeErr := errors.New("не удалось получить алерты")

	fakeSystem := &FakeSystem{
		alertsErr: fakeErr,
	}

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewHandler(fakeSystem, prometheusHandler)

	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	rw := httptest.NewRecorder()

	handler.AlertsHandler(rw, req)

	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("ожидался код ответа %v, получили %v",
			http.StatusInternalServerError, rw.Code)
	}
}

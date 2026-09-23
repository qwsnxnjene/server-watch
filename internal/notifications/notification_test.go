package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"server-watch/internal/system/model"
	"testing"
	"time"
)

type MockQueue struct {
	notification Notification
	consumeErr   error
	consumeCount int
}

func (m *MockQueue) Push(ctx context.Context, notification Notification) error {
	return nil
}

func (m *MockQueue) Consume(ctx context.Context) (Notification, error) {
	if m.consumeCount == 1 {
		return Notification{}, context.Canceled
	}
	m.consumeCount++
	return m.notification, m.consumeErr
}

type MockSender struct {
	sent    []Notification
	sentErr error
}

func (m *MockSender) Send(ctx context.Context, notification Notification) error {
	if m.sentErr != nil {
		return m.sentErr
	}

	m.sent = append(m.sent, notification)
	return nil
}

func TestWorker_Run(t *testing.T) {
	mockSender := MockSender{
		sent:    make([]Notification, 0),
		sentErr: nil,
	}

	queue := MockQueue{
		notification: Notification{
			Type:      model.AlertTypeHighCPU,
			Action:    ActionCreated,
			Value:     85,
			Threshold: 80,
			Timestamp: time.Now(),
		},
		consumeErr: nil,
	}

	worker := Worker{
		queue:   &queue,
		senders: []Sender{&mockSender},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker.Run(ctx)

	if len(mockSender.sent) != 1 {
		t.Fatalf("ожидали 1 отправленное уведомление, получили %v", len(mockSender.sent))
	}
}

func TestWorker_Run_Error(t *testing.T) {
	mockSender := MockSender{
		sent:    make([]Notification, 0),
		sentErr: errors.New("ошибка отправки"),
	}

	queue := MockQueue{
		notification: Notification{},
		consumeErr:   nil,
	}

	worker := Worker{
		queue:   &queue,
		senders: []Sender{&mockSender},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker.Run(ctx)

	if len(mockSender.sent) != 0 {
		t.Fatalf("ожидали 0 отправленных уведомлений, получили %v", len(mockSender.sent))
	}
}

func TestHttpSender_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHttpSender(http.DefaultClient, server.URL)

	err := sender.Send(context.Background(), Notification{})
	if err != nil {
		t.Fatalf("ожидали nil, получили %v", err)
	}
}

func TestHttpSender_Send_TwoRetries(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHttpSender(http.DefaultClient, server.URL)

	err := sender.Send(context.Background(), Notification{})
	if err != nil {
		t.Fatalf("ожидали nil, получили %v", err)
	}

	if attempts != 3 {
		t.Fatalf("ожидали 3 попытки, получили %d", attempts)
	}
}

func TestHttpSender_Send_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewHttpSender(http.DefaultClient, server.URL)

	err := sender.Send(context.Background(), Notification{})
	if err == nil {
		t.Fatalf("ожидали ошибку, получили %v", err)
	}
}

func TestHttpSender_Send_Post(t *testing.T) {
	notification := Notification{
		Type:      model.AlertTypeHighCPU,
		Action:    ActionResolved,
		Value:     2,
		Threshold: 10,
		Timestamp: time.Date(2026, 9, 18, 15, 14, 31, 0, time.FixedZone("MSK+3", 3*60*60)),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("ожидали POST, получили %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("ожидали Content-Type application/json, получили %s",
				r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("не удалось прочитать тело запроса: %v", err)
		}
		defer r.Body.Close()

		var got struct {
			Text string `json:"text"`
		}

		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("не удалось разобрать JSON: %v", err)
		}

		expected := formatNotification(notification)

		if got.Text != expected {
			t.Fatalf(
				"ожидали text %q, получили %q",
				expected,
				got.Text,
			)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHttpSender(http.DefaultClient, server.URL)

	err := sender.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("ожидали nil, получили %v", err)
	}
}

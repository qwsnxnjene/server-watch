package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// HttpSender отправляет уведомления через HTTP webhook
type HttpSender struct {
	client *http.Client
	url    string
}

// NewHttpSender создаёт HTTP-отправитель уведомлений
func NewHttpSender(client *http.Client, url string) *HttpSender {
	return &HttpSender{
		client: client,
		url:    url,
	}
}

type slackPayload struct {
	Text string `json:"text"`
}

// Send отправляет уведомление через HTTP webhook с повторными попытками
// для временных ошибок сети и сервера
func (h *HttpSender) Send(notification Notification) error {
	toSend := slackPayload{Text: formatNotification(notification)}
	data, err := json.Marshal(toSend)
	if err != nil {
		return fmt.Errorf("ошибка сериализации уведомления: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for attempt := range 3 {
		resp, err := h.client.Do(req)
		if err != nil {
			if attempt == 2 {
				return fmt.Errorf("ошибка запроса: %w", err)
			}

			time.Sleep(retryDelay(attempt))
			continue
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			resp.Body.Close()
			return nil
		case isRetryableStatus(resp.StatusCode):
			resp.Body.Close()

			if attempt == 2 {
				return fmt.Errorf("ошибка запроса: %d", resp.StatusCode)
			}
			time.Sleep(retryDelay(attempt))
			continue
		case resp.StatusCode >= 400 && resp.StatusCode < 500:
			resp.Body.Close()
			return fmt.Errorf("сервис для уведомлений вернул HTTP статус %d", resp.StatusCode)
		default:
			resp.Body.Close()
			return fmt.Errorf("неожиданный HTTP статус: %d", resp.StatusCode)
		}
	}
	return errors.New("неожиданное поведение функции Send")
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode >= 500 && statusCode < 600
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(1<<attempt) * time.Second
}

func formatNotification(notification Notification) string {
	if notification.Action == ActionCreated {
		return fmt.Sprintf("🚨ALERT: %v | Value: %.2f%% | Threshold: %.2f%% | Time: %v",
			notification.Type,
			notification.Value,
			notification.Threshold,
			notification.Timestamp.Format(time.RFC3339))
	}

	return fmt.Sprintf("✅RESOLVED: %v | Value: %.2f%% | Threshold: %.2f%%| Time: %v",
		notification.Type,
		notification.Value,
		notification.Threshold,
		notification.Timestamp.Format(time.RFC3339))
}

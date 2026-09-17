package notifications

import (
	"server-watch/internal/system/model"
	"time"
)

type NotificationAction string

const (
	ActionCreated  NotificationAction = "created"
	ActionResolved NotificationAction = "resolved"
)

type Notification struct {
	Type      model.AlertType    `json:"type"`
	Action    NotificationAction `json:"action"`
	Value     float64            `json:"value"`
	Threshold float64            `json:"threshold"`
	Timestamp time.Time          `json:"timestamp"`
}

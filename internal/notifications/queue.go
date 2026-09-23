package notifications

import "context"

// Queue определяет операции постановки уведомлений в очередь и их получения
type Queue interface {
	Push(ctx context.Context, notification Notification) error
	Consume(ctx context.Context) (Notification, error)
}

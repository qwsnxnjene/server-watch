package notifications

import "context"

type Queue interface {
	Push(notification Notification) error
	Consume(ctx context.Context) (Notification, error)
}

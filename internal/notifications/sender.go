package notifications

import "context"

// Sender определяет отправку уведомлений во внешнюю систему
type Sender interface {
	Send(ctx context.Context, notification Notification) error
}

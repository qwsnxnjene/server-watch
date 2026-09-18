package notifications

// Sender определяет отправку уведомлений во внешнюю систему
type Sender interface {
	Send(notification Notification) error
}

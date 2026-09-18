package notifications

type Sender interface {
	Send(notification Notification) error
}

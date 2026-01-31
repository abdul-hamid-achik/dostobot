package notify

import "context"

// Notification represents a notification message.
type Notification struct {
	Subject string
	Body    string
}

// Notifier is the interface for sending notifications.
type Notifier interface {
	// Send sends a notification.
	Send(ctx context.Context, notification Notification) error
}

package notify

import (
	"context"
	"log/slog"
)

// BlueskyNotifier sends notifications via Bluesky DM (future implementation).
// Currently a stub that just logs notifications.
type BlueskyNotifier struct {
	handle      string
	appPassword string
	toHandle    string
}

// BlueskyConfig holds configuration for Bluesky notifications.
type BlueskyConfig struct {
	Handle      string // Bot's handle
	AppPassword string // Bot's app password
	ToHandle    string // Handle to send notifications to
}

// NewBlueskyNotifier creates a new Bluesky notifier.
func NewBlueskyNotifier(cfg BlueskyConfig) *BlueskyNotifier {
	return &BlueskyNotifier{
		handle:      cfg.Handle,
		appPassword: cfg.AppPassword,
		toHandle:    cfg.ToHandle,
	}
}

// Send sends a notification.
// Currently just logs - Bluesky DM API is not yet available publicly.
func (b *BlueskyNotifier) Send(ctx context.Context, notification Notification) error {
	// Bluesky doesn't have a public DM API yet
	// For now, just log the notification
	slog.Info("notification",
		"to", b.toHandle,
		"subject", notification.Subject,
		"body", notification.Body,
	)

	// Future: Implement Bluesky DM when API is available
	// Or use alternative notification methods (email, webhook, etc.)

	return nil
}

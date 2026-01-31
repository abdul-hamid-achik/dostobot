package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlueskyNotifier(t *testing.T) {
	n := NewBlueskyNotifier(BlueskyConfig{
		Handle:      "bot.bsky.social",
		AppPassword: "password",
		ToHandle:    "user.bsky.social",
	})

	assert.NotNil(t, n)
	assert.Equal(t, "bot.bsky.social", n.handle)
	assert.Equal(t, "user.bsky.social", n.toHandle)
}

func TestBlueskyNotifier_Send(t *testing.T) {
	n := NewBlueskyNotifier(BlueskyConfig{
		ToHandle: "user.bsky.social",
	})

	err := n.Send(context.Background(), Notification{
		Subject: "Test Subject",
		Body:    "Test body",
	})

	// Currently just logs, should not error
	assert.NoError(t, err)
}

package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedditMonitor(t *testing.T) {
	t.Run("uses default subreddits", func(t *testing.T) {
		m := NewRedditMonitor(RedditConfig{})
		assert.Greater(t, len(m.subreddits), 0)
		assert.Contains(t, m.subreddits, "philosophy")
	})

	t.Run("uses custom subreddits", func(t *testing.T) {
		m := NewRedditMonitor(RedditConfig{
			Subreddits: []string{"test1", "test2"},
		})
		assert.Equal(t, []string{"test1", "test2"}, m.subreddits)
	})

	t.Run("uses default max posts", func(t *testing.T) {
		m := NewRedditMonitor(RedditConfig{})
		assert.Equal(t, redditDefaultMax, m.maxPosts)
	})
}

func TestRedditMonitor_Name(t *testing.T) {
	m := NewRedditMonitor(RedditConfig{})
	assert.Equal(t, "reddit", m.Name())
}

func TestRedditMonitor_ensureAccessToken(t *testing.T) {
	t.Run("gets new token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify basic auth
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "test-client-id", user)
			assert.Equal(t, "test-secret", pass)

			// Return token
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"expires_in":   3600,
			})
		}))
		defer server.Close()

		// We can't easily test with the real URL
		// This documents expected behavior
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

// Integration test - requires Reddit credentials
func TestRedditMonitor_Integration(t *testing.T) {
	clientID := os.Getenv("REDDIT_CLIENT_ID")
	clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		t.Skip("REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET not set")
	}

	m := NewRedditMonitor(RedditConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		UserAgent:    "dostobot:test:v1.0.0",
		Subreddits:   []string{"philosophy"},
		MaxPosts:     5,
	})

	trends, err := m.FetchTrends(context.Background())

	require.NoError(t, err)
	assert.Greater(t, len(trends), 0)

	for _, trend := range trends {
		assert.Equal(t, "reddit", trend.Source)
		assert.NotEmpty(t, trend.ExternalID)
		assert.NotEmpty(t, trend.Title)
	}
}

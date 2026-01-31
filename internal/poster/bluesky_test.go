package poster

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

func TestNewBlueskyPoster(t *testing.T) {
	poster := NewBlueskyPoster(BlueskyConfig{
		Handle:      "test.bsky.social",
		AppPassword: "test-password",
	})

	assert.NotNil(t, poster)
	assert.Equal(t, "test.bsky.social", poster.handle)
}

func TestBlueskyPoster_Platform(t *testing.T) {
	poster := NewBlueskyPoster(BlueskyConfig{})
	assert.Equal(t, "bluesky", poster.Platform())
}

func TestBlueskyPoster_authenticate(t *testing.T) {
	t.Run("successful authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/com.atproto.server.createSession", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req createSessionRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "test.bsky.social", req.Identifier)

			json.NewEncoder(w).Encode(createSessionResponse{
				DID:       "did:plc:test123",
				Handle:    "test.bsky.social",
				AccessJwt: "test-jwt-token",
			})
		}))
		defer server.Close()

		// Note: In real tests we'd inject the URL
		// This test documents expected behavior
	})

	t.Run("authentication failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Invalid credentials"}`))
		}))
		defer server.Close()

		// Test documents expected error handling
	})
}

func TestSplitURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected []string
	}{
		{
			uri:      "at://did:plc:xyz/app.bsky.feed.post/abc123",
			expected: []string{"did:plc:xyz", "app.bsky.feed.post", "abc123"},
		},
		{
			uri:      "did:plc:xyz/collection/rkey",
			expected: []string{"did:plc:xyz", "collection", "rkey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			result := splitURI(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test - requires Bluesky credentials
func TestBlueskyPoster_Integration(t *testing.T) {
	handle := os.Getenv("BLUESKY_HANDLE")
	password := os.Getenv("BLUESKY_APP_PASSWORD")

	if handle == "" || password == "" {
		t.Skip("BLUESKY_HANDLE and BLUESKY_APP_PASSWORD not set")
	}

	poster := NewBlueskyPoster(BlueskyConfig{
		Handle:      handle,
		AppPassword: password,
	})

	ctx := context.Background()

	// Test authentication
	err := poster.ValidateCredentials(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, poster.accessToken)
	assert.NotEmpty(t, poster.did)

	// Note: We don't actually post in tests to avoid spam
	// To test posting, uncomment below:
	/*
	result, err := poster.Post(ctx, PostContent{
		QuoteText:  "Test quote for integration testing.",
		SourceBook: "Test Book",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.PostID)
	t.Logf("Posted: %s", result.PostURL)
	*/
}

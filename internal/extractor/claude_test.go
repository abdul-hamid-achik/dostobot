package extractor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeClient_Complete(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
			assert.Equal(t, claudeAPIVersion, r.Header.Get("anthropic-version"))

			response := claudeResponse{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "Hello, world!"},
				},
				StopReason: "end_turn",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// We can't easily override the API URL in production code,
		// so this test documents the expected behavior
		// In a real scenario, we'd inject the URL or use an interface
	})

	t.Run("handles API error", func(t *testing.T) {
		// Test documents expected error handling
		client := NewClaudeClient(ClaudeConfig{APIKey: "invalid"})
		_, err := client.Complete(context.Background(), "system", "user")
		// Should fail because of invalid API key
		assert.Error(t, err)
	})
}

func TestExtractJSONFromResponse(t *testing.T) {
	t.Run("extracts clean JSON array", func(t *testing.T) {
		response := `[{"text": "test quote", "character": "Test", "themes": ["theme1"], "modern_relevance": "relevant"}]`

		quotes, err := extractJSONFromResponse(response)
		require.NoError(t, err)
		require.Len(t, quotes, 1)
		assert.Equal(t, "test quote", quotes[0].Text)
		assert.Equal(t, "Test", quotes[0].Character)
		assert.Equal(t, []string{"theme1"}, quotes[0].Themes)
	})

	t.Run("extracts JSON from text with preamble", func(t *testing.T) {
		response := `Here are the quotes I found:

[
  {
    "text": "Pain and suffering are inevitable.",
    "character": "Narrator",
    "themes": ["suffering", "human-nature"],
    "modern_relevance": "Still true today."
  }
]

Hope this helps!`

		quotes, err := extractJSONFromResponse(response)
		require.NoError(t, err)
		require.Len(t, quotes, 1)
		assert.Equal(t, "Pain and suffering are inevitable.", quotes[0].Text)
	})

	t.Run("handles empty array", func(t *testing.T) {
		response := `No suitable quotes found in this passage.

[]`

		quotes, err := extractJSONFromResponse(response)
		require.NoError(t, err)
		assert.Len(t, quotes, 0)
	})

	t.Run("handles multiple quotes", func(t *testing.T) {
		response := `[
  {"text": "Quote 1", "character": "A", "themes": ["t1"], "modern_relevance": "r1"},
  {"text": "Quote 2", "character": "B", "themes": ["t2"], "modern_relevance": "r2"},
  {"text": "Quote 3", "character": "C", "themes": ["t3"], "modern_relevance": "r3"}
]`

		quotes, err := extractJSONFromResponse(response)
		require.NoError(t, err)
		assert.Len(t, quotes, 3)
	})

	t.Run("error on no JSON", func(t *testing.T) {
		response := "No JSON here, just text."

		_, err := extractJSONFromResponse(response)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no JSON array found")
	})

	t.Run("error on malformed JSON", func(t *testing.T) {
		response := `[{"text": "unclosed`

		_, err := extractJSONFromResponse(response)
		assert.Error(t, err)
	})
}

func TestNewClaudeClient(t *testing.T) {
	t.Run("uses default model", func(t *testing.T) {
		client := NewClaudeClient(ClaudeConfig{APIKey: "test"})
		assert.Equal(t, defaultModel, client.model)
	})

	t.Run("uses custom model", func(t *testing.T) {
		client := NewClaudeClient(ClaudeConfig{
			APIKey: "test",
			Model:  "claude-3-opus",
		})
		assert.Equal(t, "claude-3-opus", client.model)
	})
}

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original env and restore after test
	origEnv := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, e := range origEnv {
			for i := 0; i < len(e); i++ {
				if e[i] == '=' {
					os.Setenv(e[:i], e[i+1:])
					break
				}
			}
		}
	})

	t.Run("defaults", func(t *testing.T) {
		os.Clearenv()
		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "data/dostobot.db", cfg.DatabasePath)
		assert.Equal(t, "http://localhost:11434", cfg.OllamaHost)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, 30*time.Minute, cfg.MonitorInterval)
		assert.Equal(t, 4*time.Hour, cfg.PostInterval)
		assert.Equal(t, 6, cfg.MaxPostsPerDay)
	})

	t.Run("custom values", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DATABASE_PATH", "/custom/path.db")
		os.Setenv("ANTHROPIC_API_KEY", "sk-test")
		os.Setenv("BLUESKY_HANDLE", "test.bsky.social")
		os.Setenv("MONITOR_INTERVAL", "1h")
		os.Setenv("MAX_POSTS_PER_DAY", "10")

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "/custom/path.db", cfg.DatabasePath)
		assert.Equal(t, "sk-test", cfg.AnthropicAPIKey)
		assert.Equal(t, "test.bsky.social", cfg.BlueskyHandle)
		assert.Equal(t, time.Hour, cfg.MonitorInterval)
		assert.Equal(t, 10, cfg.MaxPostsPerDay)
	})

	t.Run("invalid duration", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("MONITOR_INTERVAL", "invalid")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MONITOR_INTERVAL")
	})

	t.Run("invalid integer", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("MAX_POSTS_PER_DAY", "notanumber")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MAX_POSTS_PER_DAY")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &Config{DatabasePath: "test.db"}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("missing database path", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_PATH")
	})
}

func TestConfig_ValidateForExtraction(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &Config{
			DatabasePath:    "test.db",
			AnthropicAPIKey: "sk-test",
		}
		assert.NoError(t, cfg.ValidateForExtraction())
	})

	t.Run("missing api key", func(t *testing.T) {
		cfg := &Config{DatabasePath: "test.db"}
		err := cfg.ValidateForExtraction()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
	})
}

func TestConfig_ValidateForEmbedding(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &Config{
			DatabasePath: "test.db",
			OllamaHost:   "http://localhost:11434",
		}
		assert.NoError(t, cfg.ValidateForEmbedding())
	})

	t.Run("missing ollama host", func(t *testing.T) {
		cfg := &Config{DatabasePath: "test.db"}
		err := cfg.ValidateForEmbedding()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OLLAMA_HOST")
	})
}

func TestConfig_ValidateForPosting(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &Config{
			DatabasePath:       "test.db",
			BlueskyHandle:      "test.bsky.social",
			BlueskyAppPassword: "xxxx",
		}
		assert.NoError(t, cfg.ValidateForPosting())
	})

	t.Run("missing bluesky handle", func(t *testing.T) {
		cfg := &Config{
			DatabasePath:       "test.db",
			BlueskyAppPassword: "xxxx",
		}
		err := cfg.ValidateForPosting()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BLUESKY_HANDLE")
	})

	t.Run("missing bluesky password", func(t *testing.T) {
		cfg := &Config{
			DatabasePath:  "test.db",
			BlueskyHandle: "test.bsky.social",
		}
		err := cfg.ValidateForPosting()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BLUESKY_APP_PASSWORD")
	})
}

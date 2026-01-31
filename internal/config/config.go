package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// Database
	DatabasePath string

	// VecLite
	VecLitePath   string // Path to VecLite database (default: data/quotes.veclite)
	EmbedProvider string // Embedding provider: "ollama" or "openai" (default: ollama)

	// Anthropic API
	AnthropicAPIKey string

	// OpenAI API (for embeddings)
	OpenAIAPIKey string

	// Bluesky
	BlueskyHandle      string
	BlueskyAppPassword string

	// Reddit OAuth
	RedditClientID     string
	RedditClientSecret string
	RedditUserAgent    string

	// Ollama
	OllamaHost  string
	OllamaModel string // Ollama model for embeddings (default: nomic-embed-text)

	// Logging
	LogLevel string

	// Scheduler settings
	MonitorInterval time.Duration
	PostInterval    time.Duration
	MaxPostsPerDay  int

	// Notification settings
	NotifyHandle string
}

// Load reads configuration from environment variables.
// It automatically loads .env file if present.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		DatabasePath:       getEnv("DATABASE_PATH", "data/dostobot.db"),
		VecLitePath:        getEnv("VECLITE_PATH", "data/quotes.veclite"),
		EmbedProvider:      getEnv("EMBED_PROVIDER", "ollama"),
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		BlueskyHandle:      getEnv("BLUESKY_HANDLE", ""),
		BlueskyAppPassword: getEnv("BLUESKY_APP_PASSWORD", ""),
		RedditClientID:     getEnv("REDDIT_CLIENT_ID", ""),
		RedditClientSecret: getEnv("REDDIT_CLIENT_SECRET", ""),
		RedditUserAgent:    getEnv("REDDIT_USER_AGENT", "dostobot:v1.0.0"),
		OllamaHost:         normalizeOllamaHost(getEnv("OLLAMA_HOST", "http://localhost:11434")),
		OllamaModel:        getEnv("OLLAMA_MODEL", "nomic-embed-text"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		NotifyHandle:       getEnv("NOTIFY_HANDLE", ""),
	}

	// Parse durations
	var err error
	cfg.MonitorInterval, err = time.ParseDuration(getEnv("MONITOR_INTERVAL", "30m"))
	if err != nil {
		return nil, fmt.Errorf("invalid MONITOR_INTERVAL: %w", err)
	}

	cfg.PostInterval, err = time.ParseDuration(getEnv("POST_INTERVAL", "4h"))
	if err != nil {
		return nil, fmt.Errorf("invalid POST_INTERVAL: %w", err)
	}

	// Parse integers
	maxPosts, err := strconv.Atoi(getEnv("MAX_POSTS_PER_DAY", "6"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_POSTS_PER_DAY: %w", err)
	}
	cfg.MaxPostsPerDay = maxPosts

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.DatabasePath == "" {
		return fmt.Errorf("DATABASE_PATH is required")
	}
	return nil
}

// ValidateForExtraction checks configuration needed for quote extraction.
func (c *Config) ValidateForExtraction() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.AnthropicAPIKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required for extraction")
	}
	return nil
}

// ValidateForEmbedding checks configuration needed for embedding generation.
func (c *Config) ValidateForEmbedding() error {
	if err := c.Validate(); err != nil {
		return err
	}
	switch c.EmbedProvider {
	case "openai":
		if c.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when EMBED_PROVIDER is openai")
		}
	case "ollama", "":
		if c.OllamaHost == "" {
			return fmt.Errorf("OLLAMA_HOST is required for embedding")
		}
	default:
		return fmt.Errorf("invalid EMBED_PROVIDER: %s (must be 'ollama' or 'openai')", c.EmbedProvider)
	}
	return nil
}

// ValidateForVecLite checks configuration needed for VecLite.
func (c *Config) ValidateForVecLite() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.VecLitePath == "" {
		return fmt.Errorf("VECLITE_PATH is required")
	}
	return c.ValidateForEmbedding()
}

// ValidateForPosting checks configuration needed for posting.
func (c *Config) ValidateForPosting() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.BlueskyHandle == "" {
		return fmt.Errorf("BLUESKY_HANDLE is required for posting")
	}
	if c.BlueskyAppPassword == "" {
		return fmt.Errorf("BLUESKY_APP_PASSWORD is required for posting")
	}
	return nil
}

// ValidateForMonitoring checks configuration needed for trend monitoring.
func (c *Config) ValidateForMonitoring() error {
	if err := c.Validate(); err != nil {
		return err
	}
	// Reddit is optional, HN works without auth
	return nil
}

// ValidateForServe checks all configuration needed for serve mode.
func (c *Config) ValidateForServe() error {
	if err := c.ValidateForExtraction(); err != nil {
		return err
	}
	if err := c.ValidateForEmbedding(); err != nil {
		return err
	}
	if err := c.ValidateForPosting(); err != nil {
		return err
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// normalizeOllamaHost ensures the Ollama host has a proper URL scheme.
// This handles cases where OLLAMA_HOST is set to a bind address like "0.0.0.0"
// (used by Ollama server) instead of a client URL like "http://localhost:11434".
func normalizeOllamaHost(host string) string {
	if host == "" {
		return "http://localhost:11434"
	}

	// If it's just a bind address (0.0.0.0 or similar), use localhost instead
	if host == "0.0.0.0" || host == "0.0.0.0:11434" {
		return "http://localhost:11434"
	}

	// If it doesn't have a scheme, add http://
	if len(host) > 0 && host[0] != 'h' {
		// Check if it starts with http
		if len(host) < 4 || host[:4] != "http" {
			return "http://" + host
		}
	}

	return host
}

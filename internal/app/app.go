package app

import (
	"context"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
	"github.com/abdulachik/dostobot/internal/matcher"
	"github.com/abdulachik/dostobot/internal/monitor"
	"github.com/abdulachik/dostobot/internal/poster"
)

// App is the main application container holding all dependencies.
type App struct {
	Config  *config.Config
	Store   *db.Store
	Embedder *embedder.Embedder
	Matcher *matcher.Matcher
	Poster  poster.Poster
	Monitors []monitor.Monitor
}

// New creates a new application instance with all dependencies wired up.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	// Create database connection
	store, err := db.NewStore(ctx, cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := store.Migrate(ctx); err != nil {
		store.Close()
		return nil, err
	}

	// Create embedder
	emb := embedder.New(embedder.Config{
		Host: cfg.OllamaHost,
	})

	// Create matcher
	m := matcher.New(matcher.Config{
		Store:    store,
		Embedder: emb,
		APIKey:   cfg.AnthropicAPIKey,
	})

	// Create monitors
	monitors := []monitor.Monitor{
		monitor.NewHackerNewsMonitor(monitor.HackerNewsConfig{MaxStories: 30}),
	}

	if cfg.RedditClientID != "" && cfg.RedditClientSecret != "" {
		monitors = append(monitors, monitor.NewRedditMonitor(monitor.RedditConfig{
			ClientID:     cfg.RedditClientID,
			ClientSecret: cfg.RedditClientSecret,
			UserAgent:    cfg.RedditUserAgent,
		}))
	}

	// Create poster
	bsPoster := poster.NewBlueskyPoster(poster.BlueskyConfig{
		Handle:      cfg.BlueskyHandle,
		AppPassword: cfg.BlueskyAppPassword,
	})

	return &App{
		Config:   cfg,
		Store:    store,
		Embedder: emb,
		Matcher:  m,
		Poster:   bsPoster,
		Monitors: monitors,
	}, nil
}

// Close closes all resources.
func (a *App) Close() error {
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}

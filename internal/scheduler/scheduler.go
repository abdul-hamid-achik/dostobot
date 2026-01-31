package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/matcher"
	"github.com/abdulachik/dostobot/internal/monitor"
	"github.com/abdulachik/dostobot/internal/poster"
	"github.com/abdulachik/dostobot/internal/vectorstore"
)

// Scheduler orchestrates the periodic tasks of the bot.
type Scheduler struct {
	cfg        *config.Config
	store      *db.Store
	quoteStore *vectorstore.QuoteStore
	matcher    *matcher.Matcher
	poster     poster.Poster
	agg        *monitor.Aggregator
	health     *Health

	lastPost time.Time
}

// Config holds scheduler configuration.
type Config struct {
	Cfg   *config.Config
	Store *db.Store
}

// New creates a new scheduler.
func New(cfg Config) *Scheduler {
	// Create VecLite quote store (loads veclite.yaml config)
	quoteStore, err := vectorstore.New(vectorstore.Config{
		Path: cfg.Cfg.VecLitePath,
	})
	if err != nil {
		slog.Error("failed to create VecLite store, falling back to in-memory index", "error", err)
		quoteStore = nil
	} else {
		slog.Info("VecLite store initialized", "path", cfg.Cfg.VecLitePath, "quotes", quoteStore.Count())
	}

	// Create matcher with VecLite (or nil for legacy in-memory fallback)
	m := matcher.New(matcher.Config{
		Store:      cfg.Store,
		QuoteStore: quoteStore,
		APIKey:     cfg.Cfg.AnthropicAPIKey,
	})

	// Create monitors
	monitors := []monitor.Monitor{
		monitor.NewHackerNewsMonitor(monitor.HackerNewsConfig{MaxStories: 30}),
	}

	// Add Reddit if configured
	if cfg.Cfg.RedditClientID != "" && cfg.Cfg.RedditClientSecret != "" {
		monitors = append(monitors, monitor.NewRedditMonitor(monitor.RedditConfig{
			ClientID:     cfg.Cfg.RedditClientID,
			ClientSecret: cfg.Cfg.RedditClientSecret,
			UserAgent:    cfg.Cfg.RedditUserAgent,
		}))
	}

	// Create aggregator
	agg := monitor.NewAggregator(monitor.AggregatorConfig{
		Store:    cfg.Store,
		Monitors: monitors,
		Filter:   monitor.NewFilter(monitor.FilterConfig{}),
	})

	// Create poster
	bsPoster := poster.NewBlueskyPoster(poster.BlueskyConfig{
		Handle:      cfg.Cfg.BlueskyHandle,
		AppPassword: cfg.Cfg.BlueskyAppPassword,
	})

	return &Scheduler{
		cfg:        cfg.Cfg,
		store:      cfg.Store,
		quoteStore: quoteStore,
		matcher:    m,
		poster:     bsPoster,
		agg:        agg,
		health:     NewHealth(),
	}
}

// Close releases resources held by the scheduler.
func (s *Scheduler) Close() error {
	if s.quoteStore != nil {
		return s.quoteStore.Close()
	}
	return nil
}

// Run starts the scheduler main loop.
func (s *Scheduler) Run(ctx context.Context) error {
	slog.Info("starting scheduler",
		"monitor_interval", s.cfg.MonitorInterval,
		"post_interval", s.cfg.PostInterval,
		"max_posts_per_day", s.cfg.MaxPostsPerDay,
	)

	// Validate credentials on startup
	if err := s.poster.ValidateCredentials(ctx); err != nil {
		s.health.SetUnhealthy("bluesky", err)
		slog.Error("failed to validate Bluesky credentials", "error", err)
	} else {
		s.health.SetHealthy("bluesky", "authenticated")
	}

	// Load the vector index on startup
	if err := s.matcher.LoadIndex(ctx); err != nil {
		s.health.SetUnhealthy("index", err)
		slog.Error("failed to load vector index", "error", err)
	} else {
		s.health.SetHealthy("index", "loaded")
	}

	// Create tickers
	monitorTicker := time.NewTicker(s.cfg.MonitorInterval)
	postTicker := time.NewTicker(s.cfg.PostInterval)
	defer monitorTicker.Stop()
	defer postTicker.Stop()

	// Run initial monitoring
	s.runMonitorCycle(ctx)

	// Main loop
	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler shutting down")
			return ctx.Err()

		case <-monitorTicker.C:
			s.runMonitorCycle(ctx)

		case <-postTicker.C:
			s.runPostCycle(ctx)
		}
	}
}

// runMonitorCycle fetches and stores new trends.
func (s *Scheduler) runMonitorCycle(ctx context.Context) {
	slog.Debug("running monitor cycle")

	newTrends, err := s.agg.FetchAndStore(ctx)
	if err != nil {
		s.health.SetUnhealthy("monitor", err)
		slog.Error("monitor cycle failed", "error", err)
		return
	}

	s.health.SetHealthy("monitor", "fetched trends")
	slog.Info("monitor cycle complete", "new_trends", len(newTrends))
}

// runPostCycle attempts to post a quote.
func (s *Scheduler) runPostCycle(ctx context.Context) {
	slog.Debug("running post cycle")

	// Check daily post limit
	postsToday, err := s.store.CountPostsToday(ctx, "bluesky")
	if err != nil {
		slog.Error("failed to count today's posts", "error", err)
	} else if postsToday >= int64(s.cfg.MaxPostsPerDay) {
		slog.Info("daily post limit reached", "posts_today", postsToday, "max", s.cfg.MaxPostsPerDay)
		return
	}

	// Get unmatched trends
	unmatchedTrends, err := s.agg.GetUnmatchedTrends(ctx, 10)
	if err != nil {
		s.health.SetUnhealthy("post", err)
		slog.Error("failed to get unmatched trends", "error", err)
		return
	}

	if len(unmatchedTrends) == 0 {
		slog.Debug("no unmatched trends to post about")
		return
	}

	// Try to find a good match
	var bestMatch *matcher.MatchResult
	for _, trend := range unmatchedTrends {
		result, err := s.matcher.Match(ctx, trend)
		if err != nil {
			slog.Debug("match failed", "trend", trend.Title, "error", err)
			continue
		}

		if result != nil {
			bestMatch = result
			break
		}

		// Mark trends that don't match as skipped
		if err := s.store.UpdateTrendSkipped(ctx, db.UpdateTrendSkippedParams{
			ID:         trend.ID,
			SkipReason: sql.NullString{String: "no suitable quote match", Valid: true},
		}); err != nil {
			slog.Warn("failed to mark trend as skipped", "error", err)
		}
	}

	if bestMatch == nil {
		slog.Debug("no suitable quote-trend match found")
		return
	}

	// Format and post
	character := ""
	if bestMatch.Quote.Character.Valid {
		character = bestMatch.Quote.Character.String
	}
	formatted := poster.FormatQuote(bestMatch.Quote.Text, bestMatch.Quote.SourceBook, character)

	result, err := s.poster.Post(ctx, poster.PostContent{
		Text:       formatted,
		QuoteText:  bestMatch.Quote.Text,
		SourceBook: bestMatch.Quote.SourceBook,
		TrendTitle: bestMatch.Trend.Title,
	})
	if err != nil {
		s.health.SetUnhealthy("post", err)
		slog.Error("failed to post", "error", err)
		return
	}

	s.health.SetHealthy("post", "posted successfully")
	s.lastPost = time.Now()

	slog.Info("posted quote",
		"url", result.PostURL,
		"trend", bestMatch.Trend.Title,
		"similarity", bestMatch.VectorSimilarity,
	)

	// Record the post
	trendHash := monitor.HashTrend(monitor.Trend{
		Source:     bestMatch.Trend.Source,
		ExternalID: bestMatch.Trend.ExternalID.String,
		Title:      bestMatch.Trend.Title,
	})

	_, err = s.store.CreatePost(ctx, db.CreatePostParams{
		QuoteID:            bestMatch.Quote.ID,
		Platform:           "bluesky",
		PlatformPostID:     sql.NullString{String: result.PostID, Valid: true},
		PostUrl:            sql.NullString{String: result.PostURL, Valid: true},
		TrendID:            sql.NullInt64{Int64: bestMatch.Trend.ID, Valid: true},
		TrendTitle:         bestMatch.Trend.Title,
		TrendSource:        bestMatch.Trend.Source,
		TrendHash:          trendHash,
		RelevanceScore:     bestMatch.RelevanceScore,
		RelevanceReasoning: sql.NullString{String: bestMatch.Reasoning, Valid: bestMatch.Reasoning != ""},
		VectorSimilarity:   float64(bestMatch.VectorSimilarity),
	})
	if err != nil {
		slog.Warn("failed to record post", "error", err)
	}

	// Mark trend as matched
	if err := s.store.UpdateTrendMatched(ctx, bestMatch.Trend.ID); err != nil {
		slog.Warn("failed to mark trend as matched", "error", err)
	}

	// Update quote posted count
	if err := s.store.UpdateQuotePosted(ctx, bestMatch.Quote.ID); err != nil {
		slog.Warn("failed to update quote posted count", "error", err)
	}
}

// Health returns the health tracker.
func (s *Scheduler) Health() *Health {
	return s.health
}

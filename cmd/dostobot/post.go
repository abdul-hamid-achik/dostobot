package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
	"github.com/abdulachik/dostobot/internal/matcher"
	"github.com/abdulachik/dostobot/internal/monitor"
	"github.com/abdulachik/dostobot/internal/poster"
	"github.com/abdulachik/dostobot/internal/vectorstore"
	"github.com/spf13/cobra"
)

var postDryRun bool

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Post a quote",
	Long: `Find a matching quote for current trends and post it to Bluesky.

Examples:
  dostobot post            # Actually post
  dostobot post --dry-run  # Show what would be posted without posting`,
	RunE: runPost,
}

func init() {
	postCmd.Flags().BoolVar(&postDryRun, "dry-run", false, "Show what would be posted without actually posting")
	rootCmd.AddCommand(postCmd)
}

func runPost(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !postDryRun {
		if err := cfg.ValidateForPosting(); err != nil {
			return fmt.Errorf("validate config: %w", err)
		}
	}

	if err := cfg.ValidateForEmbedding(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	store, err := db.NewStore(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("starting post workflow", "dry_run", postDryRun)

	// Create embedder (for fallback)
	emb := embedder.New(embedder.Config{
		Host: cfg.OllamaHost,
	})

	// Create VecLite store if configured
	var quoteStore *vectorstore.QuoteStore
	if cfg.VecLitePath != "" {
		quoteStore, err = vectorstore.New(vectorstore.Config{
			Path: cfg.VecLitePath,
		})
		if err != nil {
			slog.Warn("failed to open VecLite, falling back to in-memory", "error", err)
		} else {
			defer quoteStore.Close()
			slog.Info("using VecLite for search", "documents", quoteStore.Count())
		}
	}

	// Create matcher
	m := matcher.New(matcher.Config{
		Store:      store,
		Embedder:   emb,
		QuoteStore: quoteStore,
		APIKey:     cfg.AnthropicAPIKey,
	})

	// Monitor for trends
	slog.Info("fetching trends")
	hnMonitor := monitor.NewHackerNewsMonitor(monitor.HackerNewsConfig{MaxStories: 20})

	monitors := []monitor.Monitor{hnMonitor}

	// Add Reddit if configured
	if cfg.RedditClientID != "" && cfg.RedditClientSecret != "" {
		redditMonitor := monitor.NewRedditMonitor(monitor.RedditConfig{
			ClientID:     cfg.RedditClientID,
			ClientSecret: cfg.RedditClientSecret,
			UserAgent:    cfg.RedditUserAgent,
		})
		monitors = append(monitors, redditMonitor)
	}

	// Aggregate trends
	agg := monitor.NewAggregator(monitor.AggregatorConfig{
		Store:    store,
		Monitors: monitors,
		Filter:   monitor.NewFilter(monitor.FilterConfig{}),
	})

	newTrends, err := agg.FetchAndStore(ctx)
	if err != nil {
		slog.Warn("trend aggregation had errors", "error", err)
	}

	slog.Info("fetched trends", "new", len(newTrends))

	// Get unmatched trends
	unmatchedTrends, err := agg.GetUnmatchedTrends(ctx, 10)
	if err != nil {
		return fmt.Errorf("get unmatched trends: %w", err)
	}

	if len(unmatchedTrends) == 0 {
		fmt.Println("No unmatched trends found.")
		return nil
	}

	// Try to match each trend
	var bestMatch *matcher.MatchResult
	for _, trend := range unmatchedTrends {
		slog.Debug("trying to match trend", "title", trend.Title)

		result, err := m.Match(ctx, trend)
		if err != nil {
			slog.Warn("match failed", "trend", trend.Title, "error", err)
			continue
		}

		if result != nil {
			bestMatch = result
			break
		}
	}

	if bestMatch == nil {
		fmt.Println("No suitable quote-trend match found.")
		return nil
	}

	// Format the post
	character := ""
	if bestMatch.Quote.Character.Valid {
		character = bestMatch.Quote.Character.String
	}
	formatted := poster.FormatQuote(bestMatch.Quote.Text, bestMatch.Quote.SourceBook, character)

	// Display what we're posting
	fmt.Println()
	fmt.Println("=== Post Content ===")
	fmt.Println()
	fmt.Println(formatted)
	fmt.Println()
	fmt.Printf("Trend: %s\n", bestMatch.Trend.Title)
	fmt.Printf("Similarity: %.2f\n", bestMatch.VectorSimilarity)
	fmt.Printf("Relevance: %.2f\n", bestMatch.RelevanceScore)
	fmt.Println()

	if postDryRun {
		fmt.Println("=== DRY RUN - Not posting ===")
		return nil
	}

	// Actually post
	bsPoster := poster.NewBlueskyPoster(poster.BlueskyConfig{
		Handle:      cfg.BlueskyHandle,
		AppPassword: cfg.BlueskyAppPassword,
	})

	result, err := bsPoster.Post(ctx, poster.PostContent{
		Text:       formatted,
		QuoteText:  bestMatch.Quote.Text,
		SourceBook: bestMatch.Quote.SourceBook,
		TrendTitle: bestMatch.Trend.Title,
	})
	if err != nil {
		return fmt.Errorf("post to Bluesky: %w", err)
	}

	fmt.Printf("Posted successfully!\nURL: %s\n", result.PostURL)

	// Record the post
	trendHash := monitor.HashTrend(monitor.Trend{
		Source:     bestMatch.Trend.Source,
		ExternalID: bestMatch.Trend.ExternalID.String,
		Title:      bestMatch.Trend.Title,
	})

	_, err = store.CreatePost(ctx, db.CreatePostParams{
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
	if err := store.UpdateTrendMatched(ctx, bestMatch.Trend.ID); err != nil {
		slog.Warn("failed to mark trend as matched", "error", err)
	}

	// Update quote posted count
	if err := store.UpdateQuotePosted(ctx, bestMatch.Quote.ID); err != nil {
		slog.Warn("failed to update quote posted count", "error", err)
	}

	return nil
}

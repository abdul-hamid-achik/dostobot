package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
	"github.com/abdulachik/dostobot/internal/matcher"
	"github.com/abdulachik/dostobot/internal/vectorstore"
	"github.com/spf13/cobra"
)

var matchCmd = &cobra.Command{
	Use:   "match [trend]",
	Short: "Test matching with a trend",
	Long: `Test the quote matching system with a given trend string.

Example:
  dostobot match "Political scandal shakes the nation"`,
	Args: cobra.ExactArgs(1),
	RunE: runMatch,
}

func init() {
	rootCmd.AddCommand(matchCmd)
}

func runMatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	trendText := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.ValidateForExtraction(); err != nil {
		return fmt.Errorf("validate config: %w", err)
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

	slog.Info("matching trend", "trend", trendText)

	// Create embedder
	emb := embedder.New(embedder.Config{
		Host:  cfg.OllamaHost,
		Model: cfg.OllamaModel,
	})

	// Check if VecLite is available
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

	// Match the text
	result, err := m.MatchText(ctx, trendText)
	if err != nil {
		return fmt.Errorf("match: %w", err)
	}

	if result == nil {
		fmt.Println("No matching quote found.")
		return nil
	}

	fmt.Println()
	fmt.Println("=== Matched Quote ===")
	fmt.Printf("Similarity: %.2f\n", result.VectorSimilarity)
	fmt.Println()
	fmt.Printf("\"%s\"\n", result.Quote.Text)
	fmt.Println()
	fmt.Printf("— %s\n", result.Quote.SourceBook)

	if result.Quote.Character.Valid && result.Quote.Character.String != "" {
		fmt.Printf("  (Character: %s)\n", result.Quote.Character.String)
	}

	fmt.Printf("\nThemes: %s\n", result.Quote.Themes)

	return nil
}

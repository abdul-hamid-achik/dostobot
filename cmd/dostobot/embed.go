package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/vectorstore"
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Generate embeddings for quotes",
	Long: `Generate vector embeddings for quotes and store them in VecLite.

Uses the embedding provider configured in veclite.yaml:
  - openai: OpenAI API (requires OPENAI_API_KEY env var)
  - ollama: Local Ollama server`,
	RunE: runEmbed,
}

func init() {
	rootCmd.AddCommand(embedCmd)
}

func runEmbed(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
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

	// Create VecLite store (uses veclite.yaml for embedder config)
	quoteStore, err := vectorstore.New(vectorstore.Config{
		Path: cfg.VecLitePath,
	})
	if err != nil {
		return fmt.Errorf("create quote store: %w", err)
	}
	defer quoteStore.Close()

	// Get all quotes from SQLite
	quotes, err := store.ListQuotes(ctx, db.ListQuotesParams{
		Limit:  100000, // Get all quotes
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("list quotes: %w", err)
	}

	existingCount := quoteStore.Count()
	needEmbed := len(quotes) - existingCount

	if needEmbed <= 0 {
		slog.Info("all quotes already embedded",
			"total", len(quotes),
			"in_veclite", existingCount,
		)
		return nil
	}

	slog.Info("embedding quotes",
		"total", len(quotes),
		"already_embedded", existingCount,
		"need_embedding", needEmbed,
	)

	// Embed quotes that aren't in VecLite yet
	// For simplicity, we'll re-embed all since we can't easily check which are missing
	// VecLite handles duplicates gracefully
	start := time.Now()
	embedded := 0
	errors := 0

	for i, q := range quotes {
		_, err := quoteStore.InsertQuote(ctx, q)
		if err != nil {
			slog.Warn("failed to embed quote", "id", q.ID, "error", err)
			errors++
			continue
		}

		embedded++
		if embedded%100 == 0 {
			elapsed := time.Since(start)
			rate := float64(embedded) / elapsed.Seconds()
			slog.Info("progress",
				"embedded", embedded,
				"total", len(quotes),
				"rate", fmt.Sprintf("%.1f/sec", rate),
			)
		}

		// Sync periodically
		if (i+1)%500 == 0 {
			if err := quoteStore.Sync(); err != nil {
				slog.Warn("failed to sync", "error", err)
			}
		}
	}

	// Final sync
	if err := quoteStore.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	elapsed := time.Since(start)
	slog.Info("embedding complete",
		"embedded", embedded,
		"errors", errors,
		"duration", elapsed.Round(time.Second),
		"rate", fmt.Sprintf("%.1f/sec", float64(embedded)/elapsed.Seconds()),
	)

	return nil
}

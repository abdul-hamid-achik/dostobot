package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/vectorstore"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long:  `Display statistics about quotes, posts, and trends in the database.`,
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
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

	// Ensure migrations are run
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Get quote stats
	totalQuotes, err := store.CountQuotes(ctx)
	if err != nil {
		return fmt.Errorf("count quotes: %w", err)
	}

	quotesWithEmbeddings, err := store.CountQuotesWithEmbeddings(ctx)
	if err != nil {
		return fmt.Errorf("count quotes with embeddings: %w", err)
	}

	quotesByBook, err := store.CountQuotesByBook(ctx)
	if err != nil {
		return fmt.Errorf("count quotes by book: %w", err)
	}

	// Get post count
	var totalPosts int64
	err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts").Scan(&totalPosts)
	if err != nil {
		slog.Warn("failed to count posts", "error", err)
	}

	// Get trend count
	var totalTrends int64
	err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM trends").Scan(&totalTrends)
	if err != nil {
		slog.Warn("failed to count trends", "error", err)
	}

	// Print stats
	fmt.Println("=== DostoBot Statistics ===")
	fmt.Println()
	fmt.Printf("Database: %s\n", cfg.DatabasePath)
	fmt.Println()
	fmt.Println("Quotes:")
	fmt.Printf("  Total: %d\n", totalQuotes)
	fmt.Printf("  With embeddings: %d\n", quotesWithEmbeddings)
	fmt.Printf("  Without embeddings: %d\n", totalQuotes-quotesWithEmbeddings)
	fmt.Println()

	if len(quotesByBook) > 0 {
		fmt.Println("  By book:")
		for _, row := range quotesByBook {
			fmt.Printf("    %s: %d\n", row.SourceBook, row.Count)
		}
		fmt.Println()
	}

	fmt.Println("Activity:")
	fmt.Printf("  Total posts: %d\n", totalPosts)
	fmt.Printf("  Total trends tracked: %d\n", totalTrends)
	fmt.Println()

	// Check VecLite stats if configured
	if cfg.VecLitePath != "" {
		quoteStore, err := vectorstore.New(vectorstore.Config{
			Path: cfg.VecLitePath,
		})
		if err != nil {
			slog.Warn("failed to open VecLite", "error", err)
		} else {
			defer quoteStore.Close()
			stats := quoteStore.Stats()
			fmt.Println("VecLite:")
			fmt.Printf("  Path: %s\n", cfg.VecLitePath)
			fmt.Printf("  Documents: %d\n", stats.Count)
			fmt.Printf("  Dimension: %d\n", stats.Dimension)
			fmt.Printf("  Distance: %s\n", stats.DistanceType)
			fmt.Printf("  Index: %s\n", stats.IndexType)
			fmt.Println()
		}
	}

	return nil
}

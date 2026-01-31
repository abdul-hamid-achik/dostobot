package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/extractor"
	"github.com/spf13/cobra"
)

var (
	extractAll  bool
	extractBook string
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract quotes from books",
	Long: `Extract memorable quotes from Dostoyevsky books using Claude AI.

Examples:
  dostobot extract --all                    # Extract from all books
  dostobot extract --book "Crime and Punishment"  # Extract from specific book`,
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().BoolVar(&extractAll, "all", false, "Extract from all books")
	extractCmd.Flags().StringVar(&extractBook, "book", "", "Extract from specific book")
	rootCmd.AddCommand(extractCmd)
}

func runExtract(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.ValidateForExtraction(); err != nil {
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

	if !extractAll && extractBook == "" {
		return fmt.Errorf("must specify --all or --book")
	}

	slog.Info("starting quote extraction",
		"all", extractAll,
		"book", extractBook,
	)

	ext := extractor.New(extractor.Config{
		Store:    store,
		APIKey:   cfg.AnthropicAPIKey,
		BooksDir: "books",
	})

	if extractAll {
		return ext.ExtractAll(ctx)
	}
	return ext.ExtractBook(ctx, extractBook)
}

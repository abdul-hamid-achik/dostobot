package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  `Run all pending database migrations to set up or update the schema.`,
	RunE:  runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	slog.Info("connecting to database", "path", cfg.DatabasePath)
	store, err := db.NewStore(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("migrations completed successfully")
	return nil
}

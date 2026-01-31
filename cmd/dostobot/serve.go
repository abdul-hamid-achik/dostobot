package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdulachik/dostobot/internal/config"
	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/scheduler"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the bot daemon",
	Long: `Run the DostoBot daemon that monitors trends, matches quotes,
and posts to social media on a schedule.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.ValidateForServe(); err != nil {
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

	slog.Info("starting DostoBot daemon",
		"monitor_interval", cfg.MonitorInterval,
		"post_interval", cfg.PostInterval,
		"max_posts_per_day", cfg.MaxPostsPerDay,
	)

	// Create and start the scheduler
	sched := scheduler.New(scheduler.Config{
		Cfg:   cfg,
		Store: store,
	})
	defer sched.Close()

	// Run scheduler in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- sched.Run(ctx)
	}()

	// Wait for shutdown signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			return fmt.Errorf("scheduler error: %w", err)
		}
	}

	slog.Info("shutting down...")
	cancel()

	return nil
}

package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dostobot",
	Short: "A Dostoyevsky quote bot for social media",
	Long: `DostoBot is an intelligent social media bot that posts
contextually relevant Dostoyevsky quotes to Bluesky based on trending topics.`,
}

func init() {
	// Load .env file if present
	_ = godotenv.Load()

	// Set up logging
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

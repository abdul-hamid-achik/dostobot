package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// Book represents a Dostoyevsky book to download.
type Book struct {
	Filename string
	Title    string
	URL      string
}

// Books to download from Project Gutenberg.
var books = []Book{
	{
		Filename: "crime-and-punishment.txt",
		Title:    "Crime and Punishment",
		URL:      "https://www.gutenberg.org/cache/epub/2554/pg2554.txt",
	},
	{
		Filename: "brothers-karamazov.txt",
		Title:    "The Brothers Karamazov",
		URL:      "https://www.gutenberg.org/cache/epub/28054/pg28054.txt",
	},
	{
		Filename: "notes-from-underground.txt",
		Title:    "Notes from Underground",
		URL:      "https://www.gutenberg.org/cache/epub/600/pg600.txt",
	},
	{
		Filename: "the-idiot.txt",
		Title:    "The Idiot",
		URL:      "https://www.gutenberg.org/cache/epub/2638/pg2638.txt",
	},
	{
		Filename: "the-possessed.txt",
		Title:    "The Possessed (Demons)",
		URL:      "https://www.gutenberg.org/cache/epub/8117/pg8117.txt",
	},
	{
		Filename: "the-gambler.txt",
		Title:    "The Gambler",
		URL:      "https://www.gutenberg.org/cache/epub/2197/pg2197.txt",
	},
	{
		Filename: "poor-folk.txt",
		Title:    "Poor Folk",
		URL:      "https://www.gutenberg.org/cache/epub/2302/pg2302.txt",
	},
}

var (
	downloadForce bool
	booksDir      string
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download Dostoyevsky books from Project Gutenberg",
	Long: `Download all Dostoyevsky books from Project Gutenberg to the books/ directory.

Books downloaded:
  - Crime and Punishment
  - The Brothers Karamazov
  - Notes from Underground
  - The Idiot
  - The Possessed (Demons)
  - The Gambler
  - Poor Folk`,
	RunE: runDownload,
}

func init() {
	downloadCmd.Flags().BoolVarP(&downloadForce, "force", "f", false, "Re-download even if file exists")
	downloadCmd.Flags().StringVar(&booksDir, "dir", "books", "Directory to save books")
	rootCmd.AddCommand(downloadCmd)
}

func runDownload(cmd *cobra.Command, args []string) error {
	// Create books directory if it doesn't exist
	if err := os.MkdirAll(booksDir, 0755); err != nil {
		return fmt.Errorf("create books directory: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	fmt.Println("Downloading Dostoyevsky books from Project Gutenberg...")
	fmt.Println()

	downloaded := 0
	skipped := 0

	for _, book := range books {
		path := filepath.Join(booksDir, book.Filename)

		// Check if already exists
		if !downloadForce {
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("  ✓ %s (already downloaded)\n", book.Title)
				skipped++
				continue
			}
		}

		fmt.Printf("  ↓ Downloading %s...", book.Title)

		if err := downloadFile(cmd.Context(), client, book.URL, path); err != nil {
			fmt.Printf(" ERROR: %v\n", err)
			slog.Error("failed to download book", "title", book.Title, "error", err)
			continue
		}

		fmt.Println(" done")
		downloaded++
	}

	fmt.Println()
	fmt.Printf("Downloaded: %d, Skipped: %d\n", downloaded, skipped)
	fmt.Printf("Books saved to: %s/\n", booksDir)

	return nil
}

func downloadFile(ctx context.Context, client *http.Client, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

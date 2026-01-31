package extractor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdulachik/dostobot/internal/db"
)

// BookInfo maps file names to book titles.
var BookInfo = map[string]string{
	"crime-and-punishment.txt":  "Crime and Punishment",
	"brothers-karamazov.txt":    "The Brothers Karamazov",
	"notes-from-underground.txt": "Notes from Underground",
	"the-idiot.txt":             "The Idiot",
	"the-possessed.txt":         "The Possessed",
	"the-gambler.txt":           "The Gambler",
	"poor-folk.txt":             "Poor Folk",
}

// Extractor handles quote extraction from books.
type Extractor struct {
	store    *db.Store
	claude   *ClaudeClient
	chunker  *Chunker
	booksDir string
}

// Config holds configuration for the extractor.
type Config struct {
	Store    *db.Store
	APIKey   string
	BooksDir string
}

// New creates a new Extractor.
func New(cfg Config) *Extractor {
	return &Extractor{
		store: cfg.Store,
		claude: NewClaudeClient(ClaudeConfig{
			APIKey: cfg.APIKey,
		}),
		chunker:  NewChunker(DefaultChunkerConfig()),
		booksDir: cfg.BooksDir,
	}
}

// ExtractAll extracts quotes from all available books.
func (e *Extractor) ExtractAll(ctx context.Context) error {
	files, err := os.ReadDir(e.booksDir)
	if err != nil {
		return fmt.Errorf("read books directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".txt") {
			continue
		}

		bookTitle, ok := BookInfo[file.Name()]
		if !ok {
			slog.Warn("unknown book file, skipping", "file", file.Name())
			continue
		}

		if err := e.ExtractBook(ctx, bookTitle); err != nil {
			slog.Error("failed to extract book", "book", bookTitle, "error", err)
			// Continue with other books
		}
	}

	return nil
}

// ExtractBook extracts quotes from a specific book.
func (e *Extractor) ExtractBook(ctx context.Context, bookTitle string) error {
	// Find the file for this book
	var filePath string
	for file, title := range BookInfo {
		if title == bookTitle {
			filePath = filepath.Join(e.booksDir, file)
			break
		}
	}

	if filePath == "" {
		return fmt.Errorf("unknown book: %s", bookTitle)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("book file not found: %s (run 'task download' first)", filePath)
	}

	slog.Info("starting extraction", "book", bookTitle, "file", filePath)

	// Create extraction job
	job, err := e.store.CreateExtractionJob(ctx, db.CreateExtractionJobParams{
		BookTitle: bookTitle,
		FilePath:  filePath,
	})
	if err != nil {
		return fmt.Errorf("create extraction job: %w", err)
	}

	// Chunk the book
	chunks, err := e.chunker.ChunkFile(filePath)
	if err != nil {
		e.store.UpdateExtractionJobFailed(ctx, db.UpdateExtractionJobFailedParams{
			ID:           job.ID,
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return fmt.Errorf("chunk file: %w", err)
	}

	slog.Info("chunked book", "book", bookTitle, "chunks", len(chunks))

	// Update job with total chunks
	e.store.UpdateExtractionJobStarted(ctx, db.UpdateExtractionJobStartedParams{
		ID:          job.ID,
		TotalChunks: sql.NullInt64{Int64: int64(len(chunks)), Valid: true},
	})

	// Process each chunk
	totalQuotes := 0
	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info("processing chunk",
			"book", bookTitle,
			"chunk", i+1,
			"total", len(chunks),
			"words", chunk.WordCount,
		)

		quotes, err := e.claude.ExtractQuotes(ctx, bookTitle, chunk.Text)
		if err != nil {
			slog.Error("failed to extract quotes from chunk",
				"book", bookTitle,
				"chunk", i,
				"error", err,
			)
			continue
		}

		// Save quotes
		for _, q := range quotes {
			if err := e.saveQuote(ctx, bookTitle, chunk, q); err != nil {
				slog.Error("failed to save quote",
					"book", bookTitle,
					"error", err,
				)
				continue
			}
			totalQuotes++
		}

		// Update progress
		e.store.UpdateExtractionJobProgress(ctx, db.UpdateExtractionJobProgressParams{
			ID:              job.ID,
			ProcessedChunks: sql.NullInt64{Int64: int64(i + 1), Valid: true},
			QuotesExtracted: sql.NullInt64{Int64: int64(totalQuotes), Valid: true},
		})
	}

	// Mark job complete
	e.store.UpdateExtractionJobCompleted(ctx, job.ID)

	slog.Info("extraction complete",
		"book", bookTitle,
		"total_quotes", totalQuotes,
	)

	return nil
}

// saveQuote saves an extracted quote to the database.
func (e *Extractor) saveQuote(ctx context.Context, bookTitle string, chunk Chunk, quote ExtractedQuote) error {
	// Generate hash for deduplication
	hash := sha256.Sum256([]byte(quote.Text))
	textHash := hex.EncodeToString(hash[:])

	// Check if quote already exists
	_, err := e.store.GetQuoteByHash(ctx, textHash)
	if err == nil {
		slog.Debug("quote already exists, skipping", "hash", textHash[:8])
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("check existing quote: %w", err)
	}

	// Serialize themes to JSON
	themesJSON, err := json.Marshal(quote.Themes)
	if err != nil {
		return fmt.Errorf("marshal themes: %w", err)
	}

	// Create the quote
	_, err = e.store.CreateQuote(ctx, db.CreateQuoteParams{
		Text:       quote.Text,
		TextHash:   textHash,
		SourceBook: bookTitle,
		Chapter:    sql.NullString{String: chunk.Chapter, Valid: chunk.Chapter != ""},
		Character:  sql.NullString{String: quote.Character, Valid: quote.Character != ""},
		Themes:     string(themesJSON),
		ModernRelevance: sql.NullString{
			String: quote.ModernRelevance,
			Valid:  quote.ModernRelevance != "",
		},
		CharCount: int64(len(quote.Text)),
	})
	if err != nil {
		return fmt.Errorf("create quote: %w", err)
	}

	slog.Debug("saved quote",
		"book", bookTitle,
		"length", len(quote.Text),
		"themes", quote.Themes,
	)

	return nil
}

package extractor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	extractor := New(Config{
		Store:    store,
		APIKey:   "test-key",
		BooksDir: "books",
	})

	assert.NotNil(t, extractor)
	assert.NotNil(t, extractor.claude)
	assert.NotNil(t, extractor.chunker)
	assert.Equal(t, "books", extractor.booksDir)
}

func TestExtractor_ExtractBook_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	extractor := New(Config{
		Store:    store,
		APIKey:   "test-key",
		BooksDir: tmpDir, // Empty directory
	})

	err = extractor.ExtractBook(ctx, "Crime and Punishment")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExtractor_ExtractBook_UnknownBook(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	extractor := New(Config{
		Store:    store,
		APIKey:   "test-key",
		BooksDir: tmpDir,
	})

	err = extractor.ExtractBook(ctx, "Unknown Book Title")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown book")
}

func TestExtractor_saveQuote(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	extractor := New(Config{
		Store:    store,
		APIKey:   "test-key",
		BooksDir: tmpDir,
	})

	chunk := Chunk{
		Chapter: "Chapter I",
	}

	quote := ExtractedQuote{
		Text:            "Pain and suffering are always inevitable for a large intelligence and a deep heart.",
		Character:       "Raskolnikov",
		Themes:          []string{"suffering", "intelligence"},
		ModernRelevance: "Speaks to the burden of awareness.",
	}

	err = extractor.saveQuote(ctx, "Crime and Punishment", chunk, quote)
	require.NoError(t, err)

	// Verify it was saved
	count, err := store.CountQuotes(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify duplicate is skipped
	err = extractor.saveQuote(ctx, "Crime and Punishment", chunk, quote)
	require.NoError(t, err)

	count, err = store.CountQuotes(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count) // Still 1, duplicate skipped
}

func TestBookInfo(t *testing.T) {
	// Verify all expected books are mapped
	expectedBooks := []string{
		"crime-and-punishment.txt",
		"brothers-karamazov.txt",
		"notes-from-underground.txt",
		"the-idiot.txt",
		"the-possessed.txt",
		"the-gambler.txt",
		"poor-folk.txt",
	}

	for _, file := range expectedBooks {
		title, ok := BookInfo[file]
		assert.True(t, ok, "missing book mapping for %s", file)
		assert.NotEmpty(t, title, "empty title for %s", file)
	}
}

// Integration test - requires API key and books
func TestExtractor_ExtractBook_Integration(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	booksDir := os.Getenv("BOOKS_DIR")
	if booksDir == "" {
		booksDir = "../../books"
	}

	if _, err := os.Stat(filepath.Join(booksDir, "crime-and-punishment.txt")); os.IsNotExist(err) {
		t.Skip("books not downloaded")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	extractor := New(Config{
		Store:    store,
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		BooksDir: booksDir,
	})

	// This would actually run extraction - expensive!
	// In real integration tests, you might want to limit to first few chunks
	_ = extractor
	t.Log("Integration test scaffolded - uncomment to run actual extraction")
}

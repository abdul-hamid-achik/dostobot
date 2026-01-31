// Package vectorstore provides a VecLite-based vector store for quotes.
package vectorstore

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdul-hamid-achik/veclite"
	"github.com/abdulachik/dostobot/internal/db"
)

const (
	// Collection name for quotes
	quotesCollection = "quotes"
)

// Config holds configuration for the QuoteStore.
type Config struct {
	// Path to the VecLite database file (e.g., "data/quotes.veclite").
	Path string

	// ConfigPath is the path to veclite.yaml config file (optional).
	// If empty, searches ./veclite.yaml, ~/.veclite/config.yaml.
	ConfigPath string
}

// QuoteStore wraps VecLite for quote vector storage and search.
type QuoteStore struct {
	vecdb    *veclite.DB
	coll     *veclite.Collection
	embedder veclite.Embedder
}

// SearchResult represents a search result from the vector store.
type SearchResult struct {
	VecLiteID  uint64
	SQLiteID   int64
	Text       string
	Book       string
	Character  string
	Themes     string
	Similarity float32
}

// New creates a new QuoteStore using veclite.yaml configuration.
func New(cfg Config) (*QuoteStore, error) {
	slog.Debug("creating QuoteStore", "path", cfg.Path, "config_path", cfg.ConfigPath)

	// Load veclite config (searches ./veclite.yaml, ~/.veclite/config.yaml)
	vecliteCfg, err := veclite.LoadConfig(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load veclite config: %w", err)
	}

	slog.Info("loaded veclite config",
		"provider", vecliteCfg.Embedder.Provider,
	)

	// Create embedder from config
	embedder, err := veclite.NewEmbedderFromConfig(vecliteCfg.Embedder)
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}

	dimension := embedder.Dimension()
	slog.Debug("embedder created", "dimension", dimension)

	// Open VecLite database
	vecdb, err := veclite.Open(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open veclite db: %w", err)
	}

	// Get or create collection with HNSW index and text search
	coll, err := vecdb.CreateCollection(quotesCollection,
		veclite.WithDimension(dimension),
		veclite.WithDistanceType(veclite.DistanceCosine),
		veclite.WithHNSW(16, 200), // M=16, efConstruction=200
		veclite.WithTextIndex("themes", "text", "book", "character"),
		veclite.WithEmbedder(embedder),
	)
	if err != nil {
		// Collection might already exist, try to get it
		coll, err = vecdb.GetCollection(quotesCollection)
		if err != nil {
			vecdb.Close()
			return nil, fmt.Errorf("get collection: %w", err)
		}
	}

	return &QuoteStore{
		vecdb:    vecdb,
		coll:     coll,
		embedder: embedder,
	}, nil
}

// Close closes the VecLite database.
func (s *QuoteStore) Close() error {
	if s.vecdb != nil {
		return s.vecdb.Close()
	}
	return nil
}

// InsertQuote adds a quote to the vector store.
// Returns the VecLite record ID.
func (s *QuoteStore) InsertQuote(ctx context.Context, q *db.Quote) (uint64, error) {
	payload := map[string]any{
		"sqlite_id":  q.ID,
		"book":       q.SourceBook,
		"themes":     q.Themes,
		"char_count": q.CharCount,
		"text":       q.Text,
	}
	if q.Character.Valid {
		payload["character"] = q.Character.String
	}

	// Use InsertText which auto-embeds via the configured embedder
	id, err := s.coll.InsertText(q.Text, payload)
	if err != nil {
		return 0, fmt.Errorf("insert quote: %w", err)
	}

	return id, nil
}

// InsertQuoteWithEmbedding adds a quote with a pre-computed embedding.
func (s *QuoteStore) InsertQuoteWithEmbedding(ctx context.Context, q *db.Quote, embedding []float32) (uint64, error) {
	payload := map[string]any{
		"sqlite_id":  q.ID,
		"book":       q.SourceBook,
		"themes":     q.Themes,
		"char_count": q.CharCount,
		"text":       q.Text,
	}
	if q.Character.Valid {
		payload["character"] = q.Character.String
	}

	// Use InsertDocument with pre-computed embedding
	id, err := s.coll.InsertDocument(embedding, q.Text, payload)
	if err != nil {
		return 0, fmt.Errorf("insert quote with embedding: %w", err)
	}

	return id, nil
}

// Search finds quotes similar to the query text using vector search.
func (s *QuoteStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	results, err := s.coll.SearchText(query, veclite.TopK(k))
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return s.convertResults(results), nil
}

// SearchWithThreshold finds quotes above a similarity threshold.
func (s *QuoteStore) SearchWithThreshold(ctx context.Context, query string, threshold float32, maxResults int) ([]SearchResult, error) {
	results, err := s.coll.SearchText(query,
		veclite.TopK(maxResults),
		veclite.Threshold(threshold),
	)
	if err != nil {
		return nil, fmt.Errorf("search with threshold: %w", err)
	}

	return s.convertResults(results), nil
}

// SearchVector finds quotes similar to a query vector.
func (s *QuoteStore) SearchVector(ctx context.Context, queryVec []float32, k int) ([]SearchResult, error) {
	results, err := s.coll.Search(queryVec, veclite.TopK(k))
	if err != nil {
		return nil, fmt.Errorf("search vector: %w", err)
	}

	return s.convertResults(results), nil
}

// SearchVectorWithThreshold finds quotes above a similarity threshold using a vector.
func (s *QuoteStore) SearchVectorWithThreshold(ctx context.Context, queryVec []float32, threshold float32, maxResults int) ([]SearchResult, error) {
	results, err := s.coll.Search(queryVec,
		veclite.TopK(maxResults),
		veclite.Threshold(threshold),
	)
	if err != nil {
		return nil, fmt.Errorf("search vector with threshold: %w", err)
	}

	return s.convertResults(results), nil
}

// HybridSearch combines vector and BM25 text search using RRF fusion.
func (s *QuoteStore) HybridSearch(ctx context.Context, query string, k int, vectorWeight, textWeight float64) ([]SearchResult, error) {
	// First, embed the query
	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := s.coll.HybridSearch(queryVec, query,
		veclite.TopK(k),
		veclite.WithVectorWeight(vectorWeight),
		veclite.WithTextWeight(textWeight),
	)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	return s.convertResults(results), nil
}

// TextSearch performs BM25 full-text search on indexed fields.
func (s *QuoteStore) TextSearch(ctx context.Context, query string, k int) ([]SearchResult, error) {
	results, err := s.coll.TextSearch(query, veclite.TopK(k))
	if err != nil {
		return nil, fmt.Errorf("text search: %w", err)
	}

	return s.convertResults(results), nil
}

// SearchByBook filters search results by book.
func (s *QuoteStore) SearchByBook(ctx context.Context, query string, book string, k int) ([]SearchResult, error) {
	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := s.coll.Search(queryVec,
		veclite.TopK(k),
		veclite.WithFilter(veclite.Equal("book", book)),
	)
	if err != nil {
		return nil, fmt.Errorf("search by book: %w", err)
	}

	return s.convertResults(results), nil
}

// SearchByCharacter filters search results by character.
func (s *QuoteStore) SearchByCharacter(ctx context.Context, query string, character string, k int) ([]SearchResult, error) {
	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := s.coll.Search(queryVec,
		veclite.TopK(k),
		veclite.WithFilter(veclite.Equal("character", character)),
	)
	if err != nil {
		return nil, fmt.Errorf("search by character: %w", err)
	}

	return s.convertResults(results), nil
}

// Count returns the number of quotes in the store.
func (s *QuoteStore) Count() int {
	return s.coll.Count()
}

// Stats returns statistics about the quote store.
func (s *QuoteStore) Stats() veclite.CollectionStats {
	return s.coll.Stats()
}

// Sync persists any pending changes to disk.
func (s *QuoteStore) Sync() error {
	return s.vecdb.Sync()
}

// Embed generates an embedding for the given text.
func (s *QuoteStore) Embed(text string) ([]float32, error) {
	return s.embedder.Embed(text)
}

// convertResults converts VecLite results to SearchResults.
func (s *QuoteStore) convertResults(results []veclite.Result) []SearchResult {
	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		sr := SearchResult{
			VecLiteID:  r.Record.ID,
			Similarity: r.Score,
		}

		// Extract payload fields
		if r.Record.Payload != nil {
			if id, ok := r.Record.Payload["sqlite_id"].(int64); ok {
				sr.SQLiteID = id
			} else if id, ok := r.Record.Payload["sqlite_id"].(int); ok {
				sr.SQLiteID = int64(id)
			}
			if book, ok := r.Record.Payload["book"].(string); ok {
				sr.Book = book
			}
			if character, ok := r.Record.Payload["character"].(string); ok {
				sr.Character = character
			}
			if themes, ok := r.Record.Payload["themes"].(string); ok {
				sr.Themes = themes
			}
			if text, ok := r.Record.Payload["text"].(string); ok {
				sr.Text = text
			}
		}

		// Fall back to Content field for text
		if sr.Text == "" && r.Record.Content != "" {
			sr.Text = r.Record.Content
		}

		out = append(out, sr)
	}
	return out
}

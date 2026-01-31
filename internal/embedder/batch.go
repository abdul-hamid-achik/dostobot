package embedder

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/db"
)

// BatchEmbedder handles batch embedding operations.
type BatchEmbedder struct {
	embedder  *Embedder
	store     *db.Store
	batchSize int
}

// BatchConfig holds configuration for batch embedding.
type BatchConfig struct {
	Embedder  *Embedder
	Store     *db.Store
	BatchSize int
}

// NewBatchEmbedder creates a new batch embedder.
func NewBatchEmbedder(cfg BatchConfig) *BatchEmbedder {
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	return &BatchEmbedder{
		embedder:  cfg.Embedder,
		store:     cfg.Store,
		batchSize: batchSize,
	}
}

// EmbedAllQuotes generates embeddings for all quotes without embeddings.
func (b *BatchEmbedder) EmbedAllQuotes(ctx context.Context) error {
	// First, ping Ollama to make sure it's available
	if err := b.embedder.Ping(ctx); err != nil {
		return fmt.Errorf("ollama not available: %w", err)
	}

	// Get quotes without embeddings
	quotes, err := b.store.ListQuotesWithoutEmbeddings(ctx, 10000)
	if err != nil {
		return fmt.Errorf("list quotes: %w", err)
	}

	if len(quotes) == 0 {
		slog.Info("all quotes have embeddings")
		return nil
	}

	slog.Info("embedding quotes", "count", len(quotes))

	// Process in batches
	for i := 0; i < len(quotes); i += b.batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + b.batchSize
		if end > len(quotes) {
			end = len(quotes)
		}

		batch := quotes[i:end]
		slog.Info("processing batch",
			"batch", i/b.batchSize+1,
			"total", (len(quotes)+b.batchSize-1)/b.batchSize,
			"quotes", len(batch),
		)

		for _, quote := range batch {
			embedding, err := b.embedder.Embed(ctx, quote.Text)
			if err != nil {
				slog.Error("failed to embed quote",
					"quote_id", quote.ID,
					"error", err,
				)
				continue
			}

			// Convert to bytes and store
			data := EmbeddingToBytes(embedding)
			if err := b.store.UpdateQuoteEmbedding(ctx, db.UpdateQuoteEmbeddingParams{
				ID:        quote.ID,
				Embedding: data,
			}); err != nil {
				slog.Error("failed to store embedding",
					"quote_id", quote.ID,
					"error", err,
				)
				continue
			}

			slog.Debug("embedded quote", "quote_id", quote.ID, "length", len(embedding))
		}
	}

	slog.Info("embedding complete")
	return nil
}

// EmbedTrend generates an embedding for a trend.
func (b *BatchEmbedder) EmbedTrend(ctx context.Context, trend *db.Trend) ([]float32, error) {
	// Build text from trend
	text := trend.Title
	if trend.Description.Valid && trend.Description.String != "" {
		text += "\n\n" + trend.Description.String
	}

	embedding, err := b.embedder.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed trend: %w", err)
	}

	// Store embedding
	data := EmbeddingToBytes(embedding)
	if err := b.store.UpdateTrendEmbedding(ctx, db.UpdateTrendEmbeddingParams{
		ID:        trend.ID,
		Embedding: data,
	}); err != nil {
		return nil, fmt.Errorf("store trend embedding: %w", err)
	}

	return embedding, nil
}

// EmbedText generates an embedding for arbitrary text without storing.
func (b *BatchEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	return b.embedder.Embed(ctx, text)
}

// QuoteWithEmbedding represents a quote with its parsed embedding.
type QuoteWithEmbedding struct {
	Quote     *db.Quote
	Embedding []float32
}

// LoadAllEmbeddings loads all quotes with their embeddings into memory.
func (b *BatchEmbedder) LoadAllEmbeddings(ctx context.Context) ([]QuoteWithEmbedding, error) {
	quotes, err := b.store.ListQuotesWithEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list quotes: %w", err)
	}

	result := make([]QuoteWithEmbedding, 0, len(quotes))
	for _, quote := range quotes {
		embedding, err := BytesToEmbedding(quote.Embedding)
		if err != nil {
			slog.Warn("failed to parse embedding",
				"quote_id", quote.ID,
				"error", err,
			)
			continue
		}

		// Skip quotes with nil/empty embeddings
		if len(embedding) == 0 {
			continue
		}

		result = append(result, QuoteWithEmbedding{
			Quote:     quote,
			Embedding: embedding,
		})
	}

	slog.Debug("loaded embeddings", "count", len(result))
	return result, nil
}

// Stats returns embedding statistics.
type Stats struct {
	TotalQuotes       int64
	QuotesWithEmbed   int64
	QuotesWithoutEmbed int64
}

// GetStats returns embedding statistics.
func (b *BatchEmbedder) GetStats(ctx context.Context) (*Stats, error) {
	total, err := b.store.CountQuotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("count quotes: %w", err)
	}

	withEmbed, err := b.store.CountQuotesWithEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("count with embeddings: %w", err)
	}

	return &Stats{
		TotalQuotes:       total,
		QuotesWithEmbed:   withEmbed,
		QuotesWithoutEmbed: total - withEmbed,
	}, nil
}

// nullString creates a sql.NullString.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

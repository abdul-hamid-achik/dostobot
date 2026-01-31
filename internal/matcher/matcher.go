package matcher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
	"github.com/abdulachik/dostobot/internal/vectorstore"
)

// MatchResult contains the result of matching a trend to a quote.
type MatchResult struct {
	Quote            *db.Quote
	Trend            *db.Trend
	VectorSimilarity float32
	RelevanceScore   float64
	Reasoning        string
}

// Matcher orchestrates the quote matching process.
type Matcher struct {
	store          *db.Store
	embedder       *embedder.Embedder
	batchEmbedder  *embedder.BatchEmbedder
	selector       *Selector
	vectorIndex    *VectorIndex           // Legacy in-memory index (used if quoteStore is nil)
	quoteStore     *vectorstore.QuoteStore // VecLite-based store (preferred)
	minSimilarity  float32
	minRelevance   float64
	candidateCount int
}

// Config holds configuration for the matcher.
type Config struct {
	Store          *db.Store
	Embedder       *embedder.Embedder
	QuoteStore     *vectorstore.QuoteStore // Optional: use VecLite instead of in-memory index
	APIKey         string
	MinSimilarity  float32 // Minimum vector similarity (default: 0.5)
	MinRelevance   float64 // Minimum Claude relevance score (default: 0.6)
	CandidateCount int     // Number of vector search candidates (default: 10)
}

// New creates a new Matcher.
func New(cfg Config) *Matcher {
	minSim := cfg.MinSimilarity
	if minSim == 0 {
		// Default threshold - OpenAI embeddings typically have lower similarity scores
		// than Ollama, so we use a low threshold and rely on Claude for relevance
		minSim = 0.01
	}

	minRel := cfg.MinRelevance
	if minRel == 0 {
		minRel = 0.6
	}

	candCount := cfg.CandidateCount
	if candCount == 0 {
		candCount = 10
	}

	return &Matcher{
		store:    cfg.Store,
		embedder: cfg.Embedder,
		batchEmbedder: embedder.NewBatchEmbedder(embedder.BatchConfig{
			Embedder: cfg.Embedder,
			Store:    cfg.Store,
		}),
		selector:       NewSelector(SelectorConfig{APIKey: cfg.APIKey}),
		quoteStore:     cfg.QuoteStore,
		minSimilarity:  minSim,
		minRelevance:   minRel,
		candidateCount: candCount,
	}
}

// UseVecLite returns true if VecLite is configured.
func (m *Matcher) UseVecLite() bool {
	return m.quoteStore != nil
}

// LoadIndex loads all quote embeddings into memory.
// If VecLite is configured, this is a no-op since VecLite handles storage.
func (m *Matcher) LoadIndex(ctx context.Context) error {
	// If using VecLite, the index is already loaded from disk
	if m.quoteStore != nil {
		count := m.quoteStore.Count()
		slog.Info("using VecLite index", "quotes", count)
		return nil
	}

	// Fall back to legacy in-memory index
	slog.Info("loading in-memory vector index")

	quotesWithEmbed, err := m.batchEmbedder.LoadAllEmbeddings(ctx)
	if err != nil {
		return fmt.Errorf("load embeddings: %w", err)
	}

	m.vectorIndex = NewVectorIndex(quotesWithEmbed)
	slog.Info("vector index loaded", "quotes", m.vectorIndex.Size())

	return nil
}

// IndexSize returns the number of quotes in the index.
func (m *Matcher) IndexSize() int {
	if m.quoteStore != nil {
		return m.quoteStore.Count()
	}
	if m.vectorIndex != nil {
		return m.vectorIndex.Size()
	}
	return 0
}

// Match finds the best quote for a trend.
func (m *Matcher) Match(ctx context.Context, trend *db.Trend) (*MatchResult, error) {
	// Ensure index is loaded
	if m.quoteStore == nil && m.vectorIndex == nil {
		if err := m.LoadIndex(ctx); err != nil {
			return nil, err
		}
	}

	if m.IndexSize() == 0 {
		return nil, fmt.Errorf("no quotes in index")
	}

	// Generate embedding for trend
	trendText := trend.Title
	if trend.Description.Valid && trend.Description.String != "" {
		trendText += "\n\n" + trend.Description.String
	}

	slog.Debug("generating trend embedding", "title", trend.Title)

	var candidates []VectorMatch

	if m.quoteStore != nil {
		// Use VecLite hybrid search (vector + BM25 text search)
		// vectorWeight=1.0, textWeight=0.3 to prioritize semantic similarity
		results, err := m.quoteStore.HybridSearch(ctx, trendText, m.candidateCount, 1.0, 0.3)
		if err != nil {
			return nil, fmt.Errorf("veclite search: %w", err)
		}

		// Convert VecLite results to VectorMatch
		// We need to look up the full Quote from SQLite
		for _, r := range results {
			if r.Similarity < m.minSimilarity {
				continue
			}
			// Get full quote from SQLite
			quote, err := m.store.GetQuote(ctx, r.SQLiteID)
			if err != nil {
				slog.Warn("quote not found in SQLite", "sqlite_id", r.SQLiteID, "error", err)
				continue
			}
			candidates = append(candidates, VectorMatch{
				Quote:      quote,
				Similarity: r.Similarity,
			})
		}
	} else {
		// Use legacy in-memory index
		trendEmbed, err := m.embedder.Embed(ctx, trendText)
		if err != nil {
			return nil, fmt.Errorf("embed trend: %w", err)
		}
		candidates = m.vectorIndex.SearchWithThreshold(trendEmbed, m.minSimilarity, m.candidateCount)
	}

	if len(candidates) == 0 {
		slog.Debug("no candidates above similarity threshold",
			"trend", trend.Title,
			"threshold", m.minSimilarity,
		)
		return nil, nil
	}

	slog.Debug("found vector candidates",
		"count", len(candidates),
		"best_similarity", candidates[0].Similarity,
	)

	// Get quotes for batch evaluation
	quotes := make([]*db.Quote, len(candidates))
	for i, c := range candidates {
		quotes[i] = c.Quote
	}

	// Use Claude to select the best match
	batchResult, err := m.selector.EvaluateBatch(ctx, trend, quotes)
	if err != nil {
		return nil, fmt.Errorf("evaluate batch: %w", err)
	}

	if batchResult.BestMatchIndex < 0 || batchResult.BestMatchIndex >= len(candidates) {
		slog.Debug("no suitable match found by selector",
			"trend", trend.Title,
			"recommendation", batchResult.Recommendation,
		)
		return nil, nil
	}

	bestCandidate := candidates[batchResult.BestMatchIndex]

	// Find the evaluation for the best match
	var bestEval *QuoteEvaluation
	for _, eval := range batchResult.Evaluations {
		if eval.Index == batchResult.BestMatchIndex {
			bestEval = &eval
			break
		}
	}

	relevance := 0.0
	reasoning := batchResult.Recommendation
	if bestEval != nil {
		relevance = bestEval.Score
		if bestEval.Reasoning != "" {
			reasoning = bestEval.Reasoning
		}
	}

	// Check minimum relevance
	if relevance < m.minRelevance {
		slog.Debug("best match below relevance threshold",
			"trend", trend.Title,
			"relevance", relevance,
			"threshold", m.minRelevance,
		)
		return nil, nil
	}

	return &MatchResult{
		Quote:            bestCandidate.Quote,
		Trend:            trend,
		VectorSimilarity: bestCandidate.Similarity,
		RelevanceScore:   relevance,
		Reasoning:        reasoning,
	}, nil
}

// MatchText matches a text query (not a stored trend) to a quote.
func (m *Matcher) MatchText(ctx context.Context, text string) (*MatchResult, error) {
	// Ensure index is loaded
	if m.quoteStore == nil && m.vectorIndex == nil {
		if err := m.LoadIndex(ctx); err != nil {
			return nil, err
		}
	}

	if m.quoteStore != nil {
		// Use VecLite hybrid search for best results
		results, err := m.quoteStore.HybridSearch(ctx, text, m.candidateCount, 1.0, 0.3)
		if err != nil {
			return nil, fmt.Errorf("veclite search: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}

		best := results[0]

		// Get full quote from SQLite
		quote, err := m.store.GetQuote(ctx, best.SQLiteID)
		if err != nil {
			return nil, fmt.Errorf("get quote: %w", err)
		}

		return &MatchResult{
			Quote:            quote,
			VectorSimilarity: best.Similarity,
		}, nil
	}

	// Fall back to legacy in-memory search
	embed, err := m.embedder.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed text: %w", err)
	}

	candidates := m.vectorIndex.Search(embed, m.candidateCount)
	if len(candidates) == 0 {
		return nil, nil
	}

	best := candidates[0]
	return &MatchResult{
		Quote:            best.Quote,
		VectorSimilarity: best.Similarity,
	}, nil
}

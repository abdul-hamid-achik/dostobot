package matcher

import (
	"testing"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
	"github.com/stretchr/testify/assert"
)

func TestNewVectorIndex(t *testing.T) {
	quotes := []embedder.QuoteWithEmbedding{
		{
			Quote:     &db.Quote{ID: 1, Text: "Quote 1"},
			Embedding: []float32{1, 0, 0},
		},
		{
			Quote:     &db.Quote{ID: 2, Text: "Quote 2"},
			Embedding: []float32{0, 1, 0},
		},
	}

	index := NewVectorIndex(quotes)
	assert.Equal(t, 2, index.Size())
}

func TestVectorIndex_Search(t *testing.T) {
	quotes := []embedder.QuoteWithEmbedding{
		{
			Quote:     &db.Quote{ID: 1, Text: "Quote 1"},
			Embedding: []float32{1, 0, 0},
		},
		{
			Quote:     &db.Quote{ID: 2, Text: "Quote 2"},
			Embedding: []float32{0, 1, 0},
		},
		{
			Quote:     &db.Quote{ID: 3, Text: "Quote 3"},
			Embedding: []float32{0.7, 0.7, 0},
		},
	}

	index := NewVectorIndex(quotes)

	t.Run("finds most similar", func(t *testing.T) {
		// Search for something similar to Quote 1
		query := []float32{1, 0, 0}
		results := index.Search(query, 2)

		assert.Len(t, results, 2)
		assert.Equal(t, int64(1), results[0].Quote.ID) // Most similar to [1,0,0]
		assert.InDelta(t, 1.0, float64(results[0].Similarity), 0.01)
	})

	t.Run("handles k larger than size", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results := index.Search(query, 100)

		assert.Len(t, results, 3) // Only 3 quotes in index
	})

	t.Run("empty index", func(t *testing.T) {
		emptyIndex := NewVectorIndex(nil)
		results := emptyIndex.Search([]float32{1, 0, 0}, 5)

		assert.Nil(t, results)
	})
}

func TestVectorIndex_SearchWithThreshold(t *testing.T) {
	quotes := []embedder.QuoteWithEmbedding{
		{
			Quote:     &db.Quote{ID: 1, Text: "Quote 1"},
			Embedding: []float32{1, 0, 0},
		},
		{
			Quote:     &db.Quote{ID: 2, Text: "Quote 2"},
			Embedding: []float32{0, 1, 0},
		},
		{
			Quote:     &db.Quote{ID: 3, Text: "Quote 3"},
			Embedding: []float32{0.9, 0.1, 0},
		},
	}

	index := NewVectorIndex(quotes)

	t.Run("filters by threshold", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results := index.SearchWithThreshold(query, 0.9, 10)

		// Should include Quote 1 (similarity ~1.0) and Quote 3 (similarity ~0.99)
		// but not Quote 2 (similarity ~0)
		assert.GreaterOrEqual(t, len(results), 1)
		for _, r := range results {
			assert.GreaterOrEqual(t, r.Similarity, float32(0.9))
		}
	})

	t.Run("respects max results", func(t *testing.T) {
		query := []float32{0.5, 0.5, 0}
		results := index.SearchWithThreshold(query, 0.0, 1)

		assert.Len(t, results, 1)
	})

	t.Run("high threshold returns nothing", func(t *testing.T) {
		query := []float32{0, 0, 1} // Orthogonal to all
		results := index.SearchWithThreshold(query, 0.9, 10)

		assert.Len(t, results, 0)
	})
}

func BenchmarkVectorIndex_Search(b *testing.B) {
	// Create realistic index with 1000 quotes
	quotes := make([]embedder.QuoteWithEmbedding, 1000)
	for i := range quotes {
		emb := make([]float32, 768)
		for j := range emb {
			emb[j] = float32(i*768+j) / float32(1000*768)
		}
		quotes[i] = embedder.QuoteWithEmbedding{
			Quote:     &db.Quote{ID: int64(i)},
			Embedding: emb,
		}
	}

	index := NewVectorIndex(quotes)
	query := make([]float32, 768)
	for i := range query {
		query[i] = float32(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Search(query, 10)
	}
}

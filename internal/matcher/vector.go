package matcher

import (
	"sort"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/embedder"
)

// VectorMatch represents a quote matched by vector similarity.
type VectorMatch struct {
	Quote      *db.Quote
	Similarity float32
}

// VectorIndex holds quotes with their embeddings for in-memory search.
type VectorIndex struct {
	quotes     []*db.Quote
	embeddings [][]float32
}

// NewVectorIndex creates a new in-memory vector index.
func NewVectorIndex(quotesWithEmbed []embedder.QuoteWithEmbedding) *VectorIndex {
	quotes := make([]*db.Quote, len(quotesWithEmbed))
	embeddings := make([][]float32, len(quotesWithEmbed))

	for i, qe := range quotesWithEmbed {
		quotes[i] = qe.Quote
		// Normalize for faster cosine similarity computation
		embeddings[i] = embedder.Normalize(qe.Embedding)
	}

	return &VectorIndex{
		quotes:     quotes,
		embeddings: embeddings,
	}
}

// Search finds the top-k most similar quotes to the query embedding.
func (v *VectorIndex) Search(queryEmbed []float32, k int) []VectorMatch {
	if len(v.quotes) == 0 {
		return nil
	}

	// Normalize query embedding
	normalizedQuery := embedder.Normalize(queryEmbed)

	// Compute similarities
	type scoredQuote struct {
		index      int
		similarity float32
	}

	scores := make([]scoredQuote, len(v.quotes))
	for i, emb := range v.embeddings {
		scores[i] = scoredQuote{
			index:      i,
			similarity: embedder.CosineSimilarity(normalizedQuery, emb),
		}
	}

	// Sort by similarity (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].similarity > scores[j].similarity
	})

	// Return top-k
	if k > len(scores) {
		k = len(scores)
	}

	results := make([]VectorMatch, k)
	for i := 0; i < k; i++ {
		results[i] = VectorMatch{
			Quote:      v.quotes[scores[i].index],
			Similarity: scores[i].similarity,
		}
	}

	return results
}

// SearchWithThreshold finds quotes above a similarity threshold.
func (v *VectorIndex) SearchWithThreshold(queryEmbed []float32, threshold float32, maxResults int) []VectorMatch {
	if len(v.quotes) == 0 {
		return nil
	}

	normalizedQuery := embedder.Normalize(queryEmbed)

	var results []VectorMatch
	for i, emb := range v.embeddings {
		sim := embedder.CosineSimilarity(normalizedQuery, emb)
		if sim >= threshold {
			results = append(results, VectorMatch{
				Quote:      v.quotes[i],
				Similarity: sim,
			})
		}
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// Size returns the number of quotes in the index.
func (v *VectorIndex) Size() int {
	return len(v.quotes)
}

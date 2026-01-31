# ADR-005: In-Memory Vector Search

## Status

Accepted

## Context

We need to find semantically similar quotes for a given trend. Options include:

1. External vector database (Pinecone, Weaviate, etc.)
2. SQLite vec0 extension
3. In-memory vector search with Go

## Decision

Load all quote embeddings into memory on startup and compute cosine similarity in Go.

## Rationale

- **Simple implementation**: No external dependencies
- **Sufficient scale**: <1000 quotes × 768 floats = ~3MB memory
- **Fast enough**: Full scan with cosine similarity is <10ms for this dataset size
- **No cost**: No vector database hosting fees
- **Easy to test**: Pure Go, no external services

## Implementation

```go
type VectorIndex struct {
    quotes     []*Quote
    embeddings [][]float32  // Pre-parsed from BLOB
}

func (v *VectorIndex) Search(query []float32, k int) []Match {
    // Compute cosine similarity against all quotes
    // Return top-k matches
}
```

## Memory Usage

- 1000 quotes × 768 dimensions × 4 bytes = 3.07 MB
- Plus quote metadata: ~1MB
- Total: ~5MB (well within any VPS memory)

## Consequences

- Must reload index on startup
- All quotes loaded into memory
- Can migrate to SQLite vec0 or external vector DB if scale increases
- Linear search complexity O(n) but n is small

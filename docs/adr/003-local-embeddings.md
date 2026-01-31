# ADR-003: Local Embeddings via Ollama

## Status

Accepted

## Context

We need vector embeddings to match quotes to trends. Options include:

1. OpenAI Embeddings API
2. Cohere Embeddings API
3. Local embeddings via Ollama + nomic-embed-text

## Decision

Use Ollama with the `nomic-embed-text` model for generating embeddings.

## Rationale

- **Zero API cost**: No per-request charges for embedding generation
- **Privacy**: Text never leaves the machine
- **Already available**: Ollama likely already installed for development
- **Good quality**: nomic-embed-text provides 768-dimensional vectors with quality comparable to commercial APIs
- **Can run on VPS**: DO droplet with 1GB RAM is sufficient

## Model Details

- Model: `nomic-embed-text`
- Dimensions: 768
- Context: 8192 tokens
- Size: ~275MB

## Consequences

- Need Ollama installed and running in development
- Need Ollama available on production server
- Embedding generation is slower than API calls (but done in batch, one-time)
- Can switch to API-based embeddings later if needed (interface abstraction)

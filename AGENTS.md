# AGENTS.md - AI Agent Instructions

Instructions for AI coding assistants (Claude, Cursor, Copilot, etc.) working on DostoBot.

## Project Summary

DostoBot is a Bluesky bot that posts contextually relevant Dostoyevsky quotes based on trending topics. It monitors Hacker News, matches trends to quotes using hybrid vector search (VecLite), evaluates relevance with Claude, and posts to Bluesky.

**Tech Stack**: Go 1.21+, SQLite (pure Go), VecLite (vector DB), OpenAI embeddings, Claude API, Bluesky AT Protocol

## Architecture Overview

```
cmd/dostobot/           # CLI entry points
├── main.go             # Cobra root command
├── serve.go            # Daemon mode
├── extract.go          # Quote extraction
├── embed.go            # Generate embeddings
├── post.go             # Manual posting
└── ...

internal/
├── config/             # Environment configuration (.env loading)
├── db/                 # SQLite database
│   ├── queries.sql     # sqlc query definitions
│   ├── queries.sql.go  # Generated query methods
│   └── migrations/     # Schema migrations
├── vectorstore/        # VecLite integration
│   └── store.go        # Quote storage + hybrid search
├── extractor/          # Claude-powered quote extraction
├── matcher/            # Quote-trend matching
│   ├── matcher.go      # Orchestration (VecLite → Claude)
│   ├── selector.go     # Claude relevance evaluation
│   └── vector.go       # Legacy in-memory index (fallback)
├── monitor/            # Trend sources
│   ├── hackernews.go   # HN front page
│   ├── reddit.go       # Reddit (optional)
│   └── aggregator.go   # Combines sources
├── poster/             # Social media posting
│   ├── bluesky.go      # AT Protocol client
│   └── formatter.go    # Quote formatting
└── scheduler/          # Daemon orchestration
    └── scheduler.go    # Timer-based execution
```

## Key Files to Know

| File | Purpose |
|------|---------|
| `internal/vectorstore/store.go` | VecLite wrapper - hybrid search, embeddings |
| `internal/matcher/matcher.go` | Quote matching orchestration |
| `internal/matcher/selector.go` | Claude relevance evaluation |
| `internal/scheduler/scheduler.go` | Daemon main loop |
| `internal/extractor/claude.go` | Quote extraction prompts |
| `veclite.yaml` | Embedding provider config (OpenAI/Ollama) |
| `Taskfile.yml` | All build/deploy commands |

## Data Flow

```
1. Monitor (scheduler.go)
   └── Fetch trends from HN/Reddit every 30 minutes

2. Filter (monitor/filter.go)
   └── Remove politics, violence, NSFW topics

3. Search (vectorstore/store.go)
   └── HybridSearch: HNSW vectors + BM25 text search

4. Evaluate (matcher/selector.go)
   └── Claude scores quote-trend relevance (0-1)

5. Post (poster/bluesky.go)
   └── Format quote + publish via AT Protocol

6. Record (db/queries.sql.go)
   └── Store post, mark trend as matched
```

## Database Schema

**SQLite** (`data/dostobot.db`):
- `quotes` - Extracted quotes (text, book, character, themes)
- `trends` - Detected trends from monitors
- `posts` - Posted content linking quotes ↔ trends
- `extraction_jobs` - Book processing progress

**VecLite** (`data/quotes.veclite`):
- Vector embeddings (OpenAI text-embedding-3-small, 1536 dims)
- BM25 text index on: themes, text, book, character
- Metadata: sqlite_id, book, character, themes

## Code Conventions

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("load config: %w", err)
}
```

### Logging
```go
// Use structured logging
slog.Info("quote matched",
    "trend", trend.Title,
    "similarity", result.Similarity,
)
```

### SQL Queries
```sql
-- Add to internal/db/queries.sql with sqlc annotation
-- name: GetQuote :one
SELECT * FROM quotes WHERE id = ?;
```
Then run: `sqlc generate`

### Tests
```go
// Place next to implementation
// internal/matcher/matcher_test.go
func TestMatcher_Match(t *testing.T) { ... }
```

## Common Tasks

### Add a New CLI Command

1. Create `cmd/dostobot/<name>.go`:
```go
var exampleCmd = &cobra.Command{
    Use:   "example",
    Short: "One-line description",
    RunE:  runExample,
}

func init() {
    rootCmd.AddCommand(exampleCmd)
}

func runExample(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

### Add a Database Query

1. Add to `internal/db/queries.sql`:
```sql
-- name: GetQuotesByBook :many
SELECT * FROM quotes WHERE source_book = ? ORDER BY id;
```

2. Regenerate: `sqlc generate`

3. Use in code:
```go
quotes, err := store.GetQuotesByBook(ctx, "Crime and Punishment")
```

### Add a Trend Source

1. Create `internal/monitor/<source>.go`:
```go
type MyMonitor struct { ... }

func (m *MyMonitor) Fetch(ctx context.Context) ([]Trend, error) {
    // Fetch and return trends
}
```

2. Add to `scheduler.go` monitors list.

### Modify VecLite Search

Edit `internal/vectorstore/store.go`:
```go
// Hybrid search with custom weights
results, err := s.coll.HybridSearch(queryVec, query,
    veclite.TopK(k),
    veclite.WithVectorWeight(1.0),  // Semantic
    veclite.WithTextWeight(0.3),    // Keyword
)
```

## Environment Variables

**Required for `serve`**:
```bash
ANTHROPIC_API_KEY    # Claude API (extraction + matching)
OPENAI_API_KEY       # Embeddings (via veclite.yaml)
BLUESKY_HANDLE       # Bot account
BLUESKY_APP_PASSWORD # App password
```

**Optional**:
```bash
DATABASE_PATH=data/dostobot.db
VECLITE_PATH=data/quotes.veclite
MONITOR_INTERVAL=30m
POST_INTERVAL=4h
MAX_POSTS_PER_DAY=6
LOG_LEVEL=info
```

## Deployment

**Infrastructure**: Hetzner Cloud (Terraform)
**Configuration**: Ansible playbook
**Service**: Systemd unit

```bash
task deploy     # Full deploy (terraform + ansible)
task update     # Binary update only
task logs       # Stream production logs
task status     # Check service health
```

Files:
- `deploy/terraform/main.tf` - Hetzner server + firewall
- `deploy/ansible/playbook.yml` - Server setup
- `deploy/systemd/dostobot.service` - Service definition

## Testing

```bash
# All tests
go test ./... -v

# Specific package
go test ./internal/matcher/... -v

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Important Constraints

1. **No CGO** - Use `modernc.org/sqlite`, not `mattn/go-sqlite3`
2. **Cost Awareness** - Claude API costs ~$0.01 per match evaluation
3. **Rate Limits** - Bluesky and HN have rate limits
4. **Don't Commit**:
   - `.env` / `.env.production` (secrets)
   - `data/*.db` / `data/*.veclite` (databases)
   - `books/*.txt` (downloaded content)
   - `deploy/terraform/terraform.tfstate` (infrastructure state)

## Debugging Tips

### Check VecLite Status
```bash
./bin/dostobot stats
# Shows: quote count, embedding dimensions, index stats
```

### Test Matching
```bash
./bin/dostobot match "existential crisis"
# Shows: matched quote, similarity score, book
```

### View Production Logs
```bash
task logs
# Or: ssh root@SERVER "journalctl -u dostobot -f"
```

### Common Issues

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| "no quotes in index" | VecLite not loaded | Check VECLITE_PATH, run `embed` |
| Low similarity scores | Different embedding model | Re-embed with consistent model |
| "OPENAI_API_KEY not set" | Missing env var | Check .env and veclite.yaml |
| Posts not appearing | Rate limit or auth error | Check `task logs` |

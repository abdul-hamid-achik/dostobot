# AGENTS.md - AI Agent Instructions for DostoBot

## Project Overview

DostoBot is a Bluesky bot that posts contextually relevant Dostoyevsky quotes based on trending topics. It monitors Hacker News and Reddit for trends, matches them to quotes using vector similarity + Claude evaluation, and posts to Bluesky.

## Architecture

```
cmd/dostobot/          # CLI commands (single binary with subcommands)
internal/
  app/                 # Dependency injection container
  config/              # Environment configuration
  db/                  # SQLite database + sqlc-generated queries
  embedder/            # Ollama vector embeddings
  extractor/           # Claude-powered quote extraction
  matcher/             # Vector search + Claude selection
  monitor/             # Trend monitoring (HN, Reddit)
  notify/              # Notifications (stub)
  poster/              # Bluesky AT Protocol client
  scheduler/           # Daemon orchestration
```

## Key Design Decisions

1. **Single binary** - All commands in one binary (`dostobot serve`, `dostobot extract`, etc.)
2. **Pure Go SQLite** - Uses `modernc.org/sqlite` (no CGO) for easy cross-compilation
3. **Local embeddings** - Ollama + nomic-embed-text (768 dimensions, free)
4. **In-memory vector search** - <1000 quotes fit in ~3MB memory
5. **Bluesky first** - Twitter deferred to post-MVP but interface supports it

## Code Conventions

- **Taskfile commands are single words** - `task build`, not `task:build`
- **sqlc for SQL** - Queries in `internal/db/queries.sql`, run `sqlc generate`
- **Structured logging** - Use `log/slog` with appropriate levels
- **Error wrapping** - Use `fmt.Errorf("context: %w", err)`
- **Test files** - `*_test.go` alongside implementation files

## Common Tasks

### Adding a new CLI command
1. Create `cmd/dostobot/<command>.go`
2. Define cobra command with `Use`, `Short`, `Long`, `RunE`
3. Register in `init()` with `rootCmd.AddCommand()`

### Adding a new database query
1. Add SQL to `internal/db/queries.sql` with sqlc annotation
2. Run `sqlc generate`
3. Access via `store.<QueryName>(ctx, params)`

### Adding a new trend source
1. Create `internal/monitor/<source>.go` implementing `Monitor` interface
2. Add to monitors list in `scheduler.go` or `post.go`

### Running tests
```bash
go test ./... -v -cover
```

## Environment Variables

Required for full operation:
- `ANTHROPIC_API_KEY` - Claude API for extraction/matching
- `BLUESKY_HANDLE` - Bot's Bluesky handle
- `BLUESKY_APP_PASSWORD` - Bluesky app password
- `OLLAMA_HOST` - Ollama server (default: http://localhost:11434)

Optional:
- `REDDIT_CLIENT_ID` / `REDDIT_CLIENT_SECRET` - Reddit trend monitoring
- `DATABASE_PATH` - SQLite path (default: data/dostobot.db)

## Data Flow

```
1. Monitor: Fetch trends from HN/Reddit
     ↓
2. Filter: Remove sensitive topics
     ↓
3. Embed: Generate trend embedding via Ollama
     ↓
4. Vector Search: Find candidate quotes (cosine similarity)
     ↓
5. Claude Selection: Evaluate relevance, pick best match
     ↓
6. Format: Create post text with quote + attribution
     ↓
7. Post: Publish to Bluesky via AT Protocol
     ↓
8. Record: Store post in database, mark trend as matched
```

## Database Schema

Key tables:
- `quotes` - Extracted quotes with embeddings
- `trends` - Detected trends from monitors
- `posts` - Posted content linking quotes to trends
- `extraction_jobs` - Track book processing progress
- `config` - Runtime configuration

## Testing Locally

```bash
# Build
go build -o bin/dostobot ./cmd/dostobot

# Download books
./bin/dostobot download

# Set up database
./bin/dostobot migrate

# Extract quotes (needs ANTHROPIC_API_KEY)
./bin/dostobot extract --book "Crime and Punishment"

# Generate embeddings (needs Ollama)
./bin/dostobot embed

# Test matching
./bin/dostobot match "existential crisis in modern society"

# Dry run post
./bin/dostobot post --dry-run

# Run daemon
./bin/dostobot serve
```

## Deployment

Target: DigitalOcean droplet (~$6/month)

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bin/dostobot ./cmd/dostobot

# Deploy
scp bin/dostobot user@server:/opt/dostobot/
ssh user@server "sudo systemctl restart dostobot"
```

## Important Notes

- **Don't commit .env** - Contains secrets
- **Don't commit data/*.db** - Contains local database
- **Don't commit books/*.txt** - Downloaded from Gutenberg
- **Cost awareness** - Claude API calls cost money (~$0.01-0.05 per match)
- **Rate limits** - Bluesky/Reddit have rate limits, respect them

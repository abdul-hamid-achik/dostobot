# DostoBot

A social media bot that posts contextually relevant Dostoyevsky quotes based on trending topics. Monitors Hacker News for trends, finds semantically matching quotes using hybrid vector search, and posts to Bluesky.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Bluesky](https://img.shields.io/badge/Bluesky-@dostobot-0085ff?logo=bluesky)](https://bsky.app/profile/dostobot.bsky.social)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Features

- **Quote Extraction** - Uses Claude AI to extract meaningful quotes from Dostoyevsky novels
- **Hybrid Search** - VecLite with HNSW vector index + BM25 text search for optimal matching
- **Trend Monitoring** - Watches Hacker News (Reddit support included)
- **Smart Matching** - Claude evaluates quote-trend relevance before posting
- **Bluesky Posting** - Native AT Protocol integration

## Quick Start

```bash
# Clone and setup
git clone https://github.com/abdul-hamid-achik/dostobot
cd dostobot
cp .env.example .env
# Edit .env with your API keys

# Build and run
go build -o bin/dostobot ./cmd/dostobot
./bin/dostobot download    # Get books from Project Gutenberg
./bin/dostobot migrate     # Initialize database
./bin/dostobot extract     # Extract quotes (uses Claude API)
./bin/dostobot embed       # Generate vector embeddings
./bin/dostobot serve       # Start the bot
```

## Prerequisites

| Requirement | Purpose | How to Get |
|-------------|---------|------------|
| **Go 1.21+** | Build the binary | [go.dev/dl](https://go.dev/dl/) |
| **Anthropic API Key** | Quote extraction & matching | [console.anthropic.com](https://console.anthropic.com) |
| **OpenAI API Key** | Vector embeddings | [platform.openai.com](https://platform.openai.com) |
| **Bluesky Account** | Posting quotes | [bsky.app](https://bsky.app) |
| **Task** (optional) | Run task commands | `go install github.com/go-task/task/v3/cmd/task@latest` |

## Configuration

Create a `.env` file from the template:

```bash
cp .env.example .env
```

### Required Variables

```bash
ANTHROPIC_API_KEY=sk-ant-xxxxx    # Claude API for extraction/matching
OPENAI_API_KEY=sk-xxxxx           # OpenAI for embeddings
BLUESKY_HANDLE=yourbot.bsky.social
BLUESKY_APP_PASSWORD=xxxx-xxxx-xxxx-xxxx
```

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `data/dostobot.db` | SQLite database location |
| `VECLITE_PATH` | `data/quotes.veclite` | Vector database location |
| `MONITOR_INTERVAL` | `30m` | How often to check for trends |
| `POST_INTERVAL` | `4h` | How often to post |
| `MAX_POSTS_PER_DAY` | `6` | Daily post limit |
| `LOG_LEVEL` | `info` | Logging verbosity |

## Commands

```bash
dostobot download           # Download books from Project Gutenberg
dostobot migrate            # Run database migrations
dostobot extract [--book]   # Extract quotes from books
dostobot embed              # Generate vector embeddings
dostobot match "query"      # Test quote matching
dostobot post [--dry-run]   # Post a quote
dostobot stats              # Show database statistics
dostobot serve              # Run the bot daemon
```

### Using Task

If you have [Task](https://taskfile.dev) installed:

```bash
task dev        # Run daemon locally
task build      # Build binary
task test       # Run tests
task deploy     # Full deployment to Hetzner
task update     # Quick binary update
task logs       # View production logs
task status     # Check production status
```

## How It Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Hacker News │────▶│   Filter    │────▶│   VecLite   │
│   Trends    │     │  (topics)   │     │   Search    │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                    ┌─────────────┐     ┌──────▼──────┐
                    │   Bluesky   │◀────│   Claude    │
                    │    Post     │     │  Evaluate   │
                    └─────────────┘     └─────────────┘
```

1. **Monitor** - Fetches top stories from Hacker News every 30 minutes
2. **Filter** - Removes sensitive or off-topic trends
3. **Search** - Hybrid vector + text search finds candidate quotes
4. **Evaluate** - Claude scores quote-trend relevance (threshold: 0.6)
5. **Post** - Best match posted to Bluesky with attribution

## Deployment

Deploy to Hetzner Cloud with Terraform + Ansible:

```bash
# Set your Hetzner token
export HCLOUD_TOKEN=xxxxx

# Deploy everything
task deploy
```

This creates a CPX11 server (~$4.50/month) and configures:
- Systemd service with auto-restart
- Firewall (SSH, HTTP, HTTPS)
- All databases and configuration

### Production Commands

```bash
task logs       # Stream production logs
task status     # Check service status
task ssh        # SSH into server
task restart    # Restart service
task backup     # Backup databases locally
task update     # Deploy new binary only
```

## Project Structure

```
cmd/dostobot/       # CLI commands
internal/
  config/           # Environment configuration
  db/               # SQLite + sqlc queries
  embedder/         # Legacy Ollama embedder
  extractor/        # Claude quote extraction
  matcher/          # Vector search + Claude selection
  monitor/          # Trend sources (HN, Reddit)
  poster/           # Bluesky AT Protocol client
  scheduler/        # Daemon orchestration
  vectorstore/      # VecLite integration
deploy/
  terraform/        # Hetzner infrastructure
  ansible/          # Server configuration
  systemd/          # Service definition
docs/adr/           # Architecture decisions
```

## Architecture Decisions

See [`docs/adr/`](docs/adr/) for detailed records:

| ADR | Decision |
|-----|----------|
| [001](docs/adr/001-single-binary.md) | Single binary with subcommands |
| [002](docs/adr/002-sqlite-pure-go.md) | Pure Go SQLite (no CGO) |
| [003](docs/adr/003-local-embeddings.md) | Local embeddings via Ollama |
| [004](docs/adr/004-bluesky-primary.md) | Bluesky primary platform |
| [005](docs/adr/005-in-memory-vector.md) | In-memory vector search |

*Note: The project has evolved to use VecLite with OpenAI embeddings for production.*

## Cost Breakdown

| Service | Cost | Notes |
|---------|------|-------|
| OpenAI Embeddings | ~$0.001 | One-time for 3,339 quotes |
| Claude API | ~$2/book | Extraction (~$0.01/match) |
| Hetzner CPX11 | ~$4.50/month | 2 vCPU, 2GB RAM |
| Bluesky | Free | No API costs |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- **Live Bot**: [@dostobot.bsky.social](https://bsky.app/profile/dostobot.bsky.social)
- **VecLite**: [github.com/abdul-hamid-achik/veclite](https://github.com/abdul-hamid-achik/veclite)

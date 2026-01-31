# DostoBot

An intelligent social media bot that posts contextually relevant Dostoyevsky quotes to Bluesky based on trending topics.

## Setup

### Prerequisites

1. **Go 1.21+** - [Install Go](https://go.dev/dl/)

2. **Ollama** with the embedding model:
   ```bash
   # Install Ollama (macOS/Linux)
   curl -fsSL https://ollama.com/install.sh | sh

   # Pull the embedding model
   ollama pull nomic-embed-text

   # Verify it's running
   ollama list
   ```

3. **Task runner** (optional but recommended):
   ```bash
   go install github.com/go-task/task/v3/cmd/task@latest
   ```

### External Accounts

Before running, you need:

| Account | Where to Get | Required For |
|---------|--------------|--------------|
| **Anthropic API Key** | [console.anthropic.com](https://console.anthropic.com) | Quote extraction |
| **Bluesky Account** | [bsky.app](https://bsky.app) | Posting |
| **Bluesky App Password** | Bluesky Settings → App Passwords | Posting |
| **Reddit OAuth** (optional) | [reddit.com/prefs/apps](https://www.reddit.com/prefs/apps) | Reddit trend monitoring |

### Installation

```bash
# Clone the repository
git clone https://github.com/abdulachik/dostobot
cd dostobot

# Copy environment template and edit with your credentials
cp .env.example .env

# Build the binary
go build -o bin/dostobot ./cmd/dostobot
```

### Configuration

Edit `.env` with your credentials:

```bash
# Required for extraction
ANTHROPIC_API_KEY=sk-ant-xxxxx

# Required for posting
BLUESKY_HANDLE=yourbot.bsky.social
BLUESKY_APP_PASSWORD=xxxx-xxxx-xxxx-xxxx

# Optional - for Reddit trend monitoring
REDDIT_CLIENT_ID=xxxxx
REDDIT_CLIENT_SECRET=xxxxx
```

### First Run

```bash
# 1. Download Dostoyevsky books from Project Gutenberg
./bin/dostobot download

# 2. Set up the database
./bin/dostobot migrate

# 3. Extract quotes (this calls Claude API - costs ~$0.50-2 per book)
./bin/dostobot extract --book "Crime and Punishment"

# 4. Generate embeddings (requires Ollama running)
./bin/dostobot embed

# 5. Check stats
./bin/dostobot stats
```

### Testing

```bash
# Test quote matching
./bin/dostobot match "Political scandal shakes the nation"

# Dry run posting (doesn't actually post)
./bin/dostobot post --dry-run

# Actually post
./bin/dostobot post
```

### Running the Daemon

```bash
# Run the bot (monitors trends and posts on schedule)
./bin/dostobot serve
```

## Commands

```
dostobot download           # Download books from Project Gutenberg
dostobot migrate            # Run database migrations
dostobot extract [--all]    # Extract quotes from books
dostobot embed              # Generate embeddings
dostobot match "trend"      # Test matching
dostobot post [--dry-run]   # Post a quote
dostobot stats              # Show statistics
dostobot serve              # Run the bot daemon
```

## Task Commands

If you have `task` installed:

```bash
task download    # Download books
task migrate     # Run migrations
task extract     # Extract quotes (pass args with --)
task embed       # Generate embeddings
task match       # Test matching (pass args with --)
task post        # Post a quote
task stats       # Show stats
task dev         # Run daemon locally
task test        # Run tests
task build       # Build binary
```

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | - | Claude API key (required for extraction) |
| `BLUESKY_HANDLE` | - | Your Bluesky handle |
| `BLUESKY_APP_PASSWORD` | - | Bluesky app password |
| `REDDIT_CLIENT_ID` | - | Reddit OAuth client ID (optional) |
| `REDDIT_CLIENT_SECRET` | - | Reddit OAuth secret (optional) |
| `REDDIT_USER_AGENT` | `dostobot:v1.0.0` | Reddit API user agent |
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `DATABASE_PATH` | `data/dostobot.db` | SQLite database path |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `MONITOR_INTERVAL` | `30m` | How often to check for trends |
| `POST_INTERVAL` | `4h` | How often to post |
| `MAX_POSTS_PER_DAY` | `6` | Maximum posts per day |

## Architecture

See `docs/adr/` for architecture decision records:

- **ADR-001**: Single binary with subcommands
- **ADR-002**: SQLite with pure Go driver (no CGO)
- **ADR-003**: Local embeddings via Ollama
- **ADR-004**: Bluesky primary, Twitter deferred
- **ADR-005**: In-memory vector search

## Cost Estimates

- **Ollama**: Free (runs locally)
- **Claude API**: ~$0.50-2 per book for extraction, ~$0.01-0.05 per match evaluation
- **Bluesky**: Free
- **Reddit API**: Free
- **Hosting**: ~$6/month on DigitalOcean droplet

## License

MIT

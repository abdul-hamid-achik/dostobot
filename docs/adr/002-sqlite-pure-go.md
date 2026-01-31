# ADR-002: SQLite with Pure Go Driver

## Status

Accepted

## Context

We need a database for storing quotes, posts, and trends. Options include:

1. PostgreSQL
2. SQLite with CGO (`mattn/go-sqlite3`)
3. SQLite with pure Go (`modernc.org/sqlite`)

## Decision

Use SQLite with the `modernc.org/sqlite` pure Go driver.

## Rationale

- **No CGO required**: Simpler cross-compilation, no need for C toolchain
- **Easier CI/CD**: No special build steps for different platforms
- **Single file database**: Easy backup, deployment, and debugging
- **Sufficient for workload**: <1000 quotes, ~6 posts/day, minimal concurrent access
- **Cost effective**: No database hosting costs

## Trade-offs

- Slightly slower than CGO driver (~10-20% for most operations)
- Less mature than `mattn/go-sqlite3`
- No support for some SQLite extensions

## Consequences

- Cross-compilation works out of the box
- Binary is self-contained
- Can easily backup by copying the DB file
- WAL mode used for better concurrent read performance
- Single writer constraint (acceptable for this use case)

# ADR-001: Single Binary with Subcommands

## Status

Accepted

## Context

We need to decide how to structure the CLI for DostoBot. Options include:

1. Multiple separate binaries (dostobot-serve, dostobot-extract, etc.)
2. Single binary with subcommands (dostobot serve, dostobot extract, etc.)

## Decision

Use a single binary (`dostobot`) with cobra-style subcommands.

## Commands

```
dostobot serve              # Run the bot daemon
dostobot extract [--all]    # Extract quotes from books
dostobot embed              # Generate embeddings
dostobot match "trend"      # Test matching
dostobot post [--dry-run]   # Post a quote
dostobot stats              # Show statistics
dostobot migrate            # Run migrations only
```

## Rationale

- **Simpler deployment**: One file to copy to production
- **Shared initialization**: Config loading, DB connection setup shared across commands
- **Easier testing**: Single test suite, shared test utilities
- **Standard pattern**: Cobra is the de facto standard for Go CLIs
- **Better UX**: Tab completion, unified help system

## Consequences

- Slightly larger binary size (includes all command code)
- Must be careful about initialization order (lazy vs eager loading)
- All commands share the same version/release cycle

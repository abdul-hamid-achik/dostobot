# ADR-004: Bluesky Primary, Twitter Deferred

## Status

Accepted

## Context

We want to post quotes to social media. Options include:

1. Twitter/X only
2. Bluesky only
3. Both simultaneously
4. Bluesky first, Twitter later

## Decision

Build Bluesky integration first. Design the poster interface to support Twitter later, but defer Twitter implementation to post-MVP.

## Rationale

- **Simpler API**: Bluesky's AT Protocol is more developer-friendly
- **No approval process**: Twitter requires API approval which can take weeks
- **No API cost**: Bluesky API is free
- **Less rate limiting**: More generous limits for posting
- **Growing audience**: Bluesky is gaining momentum, good fit for literary content
- **Interface abstraction**: `Poster` interface allows easy addition of Twitter later

## Interface Design

```go
type Poster interface {
    Post(ctx context.Context, content PostContent) (*PostResult, error)
    Platform() string
}

// Implementations
type BlueskyPoster struct { ... }
type TwitterPoster struct { ... }  // Future
```

## Consequences

- Twitter audience not reached initially
- Can add Twitter support without changing core logic
- Need Bluesky account and app password

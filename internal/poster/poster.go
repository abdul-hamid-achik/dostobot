package poster

import (
	"context"
)

// PostContent represents the content to be posted.
type PostContent struct {
	Text       string
	QuoteText  string
	SourceBook string
	TrendTitle string
}

// PostResult represents the result of a post.
type PostResult struct {
	PostID  string
	PostURL string
}

// Poster is the interface for posting to social media platforms.
type Poster interface {
	// Platform returns the name of the platform.
	Platform() string

	// Post publishes content to the platform.
	Post(ctx context.Context, content PostContent) (*PostResult, error)

	// ValidateCredentials checks if the credentials are valid.
	ValidateCredentials(ctx context.Context) error
}

package poster

import (
	"context"
	"fmt"
)

// TwitterPoster is a stub for Twitter/X posting (post-MVP).
type TwitterPoster struct {
	// Twitter API credentials would go here
	apiKey       string
	apiSecret    string
	accessToken  string
	accessSecret string
}

// TwitterConfig holds configuration for the Twitter poster.
type TwitterConfig struct {
	APIKey       string
	APISecret    string
	AccessToken  string
	AccessSecret string
}

// NewTwitterPoster creates a new Twitter poster stub.
func NewTwitterPoster(cfg TwitterConfig) *TwitterPoster {
	return &TwitterPoster{
		apiKey:       cfg.APIKey,
		apiSecret:    cfg.APISecret,
		accessToken:  cfg.AccessToken,
		accessSecret: cfg.AccessSecret,
	}
}

// Platform returns the platform name.
func (t *TwitterPoster) Platform() string {
	return "twitter"
}

// ValidateCredentials validates Twitter credentials.
func (t *TwitterPoster) ValidateCredentials(ctx context.Context) error {
	return fmt.Errorf("Twitter posting not implemented (post-MVP)")
}

// Post publishes content to Twitter.
func (t *TwitterPoster) Post(ctx context.Context, content PostContent) (*PostResult, error) {
	return nil, fmt.Errorf("Twitter posting not implemented (post-MVP)")
}

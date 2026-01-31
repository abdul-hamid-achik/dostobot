package poster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	blueskyBaseURL = "https://bsky.social/xrpc"
)

// BlueskyPoster posts to Bluesky via the AT Protocol.
type BlueskyPoster struct {
	httpClient  *http.Client
	handle      string
	appPassword string
	accessToken string
	did         string
}

// BlueskyConfig holds configuration for the Bluesky poster.
type BlueskyConfig struct {
	Handle      string
	AppPassword string
}

// NewBlueskyPoster creates a new Bluesky poster.
func NewBlueskyPoster(cfg BlueskyConfig) *BlueskyPoster {
	return &BlueskyPoster{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		handle:      cfg.Handle,
		appPassword: cfg.AppPassword,
	}
}

// Platform returns the platform name.
func (b *BlueskyPoster) Platform() string {
	return "bluesky"
}

// createSessionRequest is the request body for session creation.
type createSessionRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// createSessionResponse is the response from session creation.
type createSessionResponse struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	AccessJwt   string `json:"accessJwt"`
	RefreshJwt  string `json:"refreshJwt"`
}

// ValidateCredentials authenticates and validates the credentials.
func (b *BlueskyPoster) ValidateCredentials(ctx context.Context) error {
	return b.authenticate(ctx)
}

func (b *BlueskyPoster) authenticate(ctx context.Context) error {
	if b.accessToken != "" {
		return nil // Already authenticated
	}

	reqBody := createSessionRequest{
		Identifier: b.handle,
		Password:   b.appPassword,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := blueskyBaseURL + "/com.atproto.server.createSession"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var session createSessionResponse
	if err := json.Unmarshal(respBody, &session); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	b.accessToken = session.AccessJwt
	b.did = session.DID

	slog.Debug("authenticated with Bluesky",
		"handle", session.Handle,
		"did", session.DID,
	)

	return nil
}

// createRecordRequest is the request body for creating a post.
type createRecordRequest struct {
	Repo       string      `json:"repo"`
	Collection string      `json:"collection"`
	Record     postRecord  `json:"record"`
}

// postRecord represents a Bluesky post.
type postRecord struct {
	Type      string    `json:"$type"`
	Text      string    `json:"text"`
	CreatedAt string    `json:"createdAt"`
	Langs     []string  `json:"langs,omitempty"`
}

// createRecordResponse is the response from creating a post.
type createRecordResponse struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// Post publishes content to Bluesky.
func (b *BlueskyPoster) Post(ctx context.Context, content PostContent) (*PostResult, error) {
	// Ensure we're authenticated
	if err := b.authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	// Format the post text
	text := content.Text
	if text == "" {
		text = FormatQuote(content.QuoteText, content.SourceBook, "")
	}

	// Check length
	if !FitsInLimit(text, BlueskyMaxLength) {
		// Truncate if needed
		attribution := fmt.Sprintf("— %s", content.SourceBook)
		truncated := TruncateQuote(content.QuoteText, BlueskyMaxLength, attribution)
		text = FormatQuote(truncated, content.SourceBook, "")
	}

	// Create the post
	record := postRecord{
		Type:      "app.bsky.feed.post",
		Text:      text,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Langs:     []string{"en"},
	}

	reqBody := createRecordRequest{
		Repo:       b.did,
		Collection: "app.bsky.feed.post",
		Record:     record,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := blueskyBaseURL + "/com.atproto.repo.createRecord"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var createResp createRecordResponse
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Construct the post URL
	// URI format: at://did:plc:xxx/app.bsky.feed.post/rkey
	// URL format: https://bsky.app/profile/handle/post/rkey
	postURL := ""
	if createResp.URI != "" {
		// Extract rkey from URI
		parts := splitURI(createResp.URI)
		if len(parts) >= 3 {
			rkey := parts[len(parts)-1]
			postURL = fmt.Sprintf("https://bsky.app/profile/%s/post/%s", b.handle, rkey)
		}
	}

	slog.Info("posted to Bluesky",
		"uri", createResp.URI,
		"url", postURL,
	)

	return &PostResult{
		PostID:  createResp.URI,
		PostURL: postURL,
	}, nil
}

// splitURI splits an AT Protocol URI into parts.
func splitURI(uri string) []string {
	// Remove at:// prefix
	if len(uri) > 5 && uri[:5] == "at://" {
		uri = uri[5:]
	}
	// Split by /
	var parts []string
	current := ""
	for _, c := range uri {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

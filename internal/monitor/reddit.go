package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	redditAuthURL    = "https://www.reddit.com/api/v1/access_token"
	redditAPIURL     = "https://oauth.reddit.com"
	redditDefaultMax = 25
)

// RedditMonitor monitors Reddit for trending posts.
type RedditMonitor struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	userAgent    string
	accessToken  string
	tokenExpiry  time.Time
	subreddits   []string
	maxPosts     int
}

// RedditConfig holds configuration for the Reddit monitor.
type RedditConfig struct {
	ClientID     string
	ClientSecret string
	UserAgent    string
	Subreddits   []string
	MaxPosts     int
}

// NewRedditMonitor creates a new Reddit monitor.
func NewRedditMonitor(cfg RedditConfig) *RedditMonitor {
	subreddits := cfg.Subreddits
	if len(subreddits) == 0 {
		// Default subreddits relevant for Dostoyevsky quotes
		subreddits = []string{
			"philosophy",
			"books",
			"literature",
			"AskPhilosophy",
			"TrueReddit",
		}
	}

	maxPosts := cfg.MaxPosts
	if maxPosts <= 0 {
		maxPosts = redditDefaultMax
	}

	return &RedditMonitor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		userAgent:    cfg.UserAgent,
		subreddits:   subreddits,
		maxPosts:     maxPosts,
	}
}

// Name returns the monitor name.
func (r *RedditMonitor) Name() string {
	return "reddit"
}

// redditListing represents a Reddit API listing response.
type redditListing struct {
	Data struct {
		Children []struct {
			Data struct {
				ID        string  `json:"id"`
				Title     string  `json:"title"`
				Selftext  string  `json:"selftext"`
				URL       string  `json:"url"`
				Permalink string  `json:"permalink"`
				Score     int     `json:"score"`
				Subreddit string  `json:"subreddit"`
				Ups       int     `json:"ups"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// FetchTrends retrieves hot posts from configured subreddits.
func (r *RedditMonitor) FetchTrends(ctx context.Context) ([]Trend, error) {
	// Ensure we have a valid access token
	if err := r.ensureAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	var allTrends []Trend

	for _, subreddit := range r.subreddits {
		trends, err := r.fetchSubredditHot(ctx, subreddit)
		if err != nil {
			slog.Warn("failed to fetch subreddit",
				"subreddit", subreddit,
				"error", err,
			)
			continue
		}
		allTrends = append(allTrends, trends...)
	}

	// Sort by score and limit
	if len(allTrends) > r.maxPosts {
		// Simple bubble sort for small list (good enough for ~100 items)
		for i := 0; i < len(allTrends)-1; i++ {
			for j := i + 1; j < len(allTrends); j++ {
				if allTrends[j].Score > allTrends[i].Score {
					allTrends[i], allTrends[j] = allTrends[j], allTrends[i]
				}
			}
		}
		allTrends = allTrends[:r.maxPosts]
	}

	slog.Debug("fetched Reddit trends", "count", len(allTrends))
	return allTrends, nil
}

func (r *RedditMonitor) ensureAccessToken(ctx context.Context) error {
	// Check if we have a valid token
	if r.accessToken != "" && time.Now().Before(r.tokenExpiry) {
		return nil
	}

	// Get new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", redditAuthURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(r.clientID, r.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Reddit auth failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	r.accessToken = tokenResp.AccessToken
	r.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	slog.Debug("obtained Reddit access token",
		"expires_in", tokenResp.ExpiresIn,
	)

	return nil
}

func (r *RedditMonitor) fetchSubredditHot(ctx context.Context, subreddit string) ([]Trend, error) {
	url := fmt.Sprintf("%s/r/%s/hot?limit=10", redditAPIURL, subreddit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+r.accessToken)
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Reddit API error (status %d): %s", resp.StatusCode, string(body))
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	trends := make([]Trend, 0, len(listing.Data.Children))
	for _, child := range listing.Data.Children {
		post := child.Data

		// Build full URL
		postURL := post.URL
		if strings.HasPrefix(post.Permalink, "/") {
			postURL = "https://www.reddit.com" + post.Permalink
		}

		trends = append(trends, Trend{
			Source:      "reddit",
			ExternalID:  post.ID,
			Title:       post.Title,
			URL:         postURL,
			Description: truncate(post.Selftext, 500),
			Score:       post.Score,
		})
	}

	return trends, nil
}

// truncate shortens a string to maxLen, adding ellipsis if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

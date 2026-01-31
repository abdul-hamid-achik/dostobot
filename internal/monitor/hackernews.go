package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	hnBaseURL     = "https://hacker-news.firebaseio.com/v0"
	hnTopStories  = "/topstories.json"
	hnItem        = "/item/%d.json"
	hnDefaultMax  = 30
)

// HackerNewsMonitor monitors Hacker News for trending stories.
type HackerNewsMonitor struct {
	httpClient *http.Client
	maxStories int
}

// HackerNewsConfig holds configuration for the HN monitor.
type HackerNewsConfig struct {
	MaxStories int
}

// NewHackerNewsMonitor creates a new Hacker News monitor.
func NewHackerNewsMonitor(cfg HackerNewsConfig) *HackerNewsMonitor {
	maxStories := cfg.MaxStories
	if maxStories <= 0 {
		maxStories = hnDefaultMax
	}

	return &HackerNewsMonitor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxStories: maxStories,
	}
}

// Name returns the monitor name.
func (h *HackerNewsMonitor) Name() string {
	return "hackernews"
}

// hnStory represents a Hacker News story.
type hnStory struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Text        string `json:"text"` // For self-posts
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"` // Comment count
	Type        string `json:"type"`
}

// FetchTrends retrieves top stories from Hacker News.
func (h *HackerNewsMonitor) FetchTrends(ctx context.Context) ([]Trend, error) {
	// Get top story IDs
	ids, err := h.fetchTopStoryIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch top stories: %w", err)
	}

	// Limit to maxStories
	if len(ids) > h.maxStories {
		ids = ids[:h.maxStories]
	}

	// Fetch story details concurrently
	stories := make([]*hnStory, len(ids))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for i, id := range ids {
		wg.Add(1)
		go func(idx int, storyID int) {
			defer wg.Done()

			story, err := h.fetchStory(ctx, storyID)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			stories[idx] = story
			mu.Unlock()
		}(i, id)
	}

	wg.Wait()

	if len(errors) > 0 {
		slog.Warn("some HN stories failed to fetch", "errors", len(errors))
	}

	// Convert to Trend
	trends := make([]Trend, 0, len(stories))
	for _, story := range stories {
		if story == nil {
			continue
		}

		// Skip non-story items
		if story.Type != "story" {
			continue
		}

		description := story.Text
		if description == "" && story.URL != "" {
			description = fmt.Sprintf("Link: %s", story.URL)
		}

		trends = append(trends, Trend{
			Source:      "hackernews",
			ExternalID:  strconv.Itoa(story.ID),
			Title:       story.Title,
			URL:         story.URL,
			Description: description,
			Score:       story.Score,
		})
	}

	slog.Debug("fetched HN trends", "count", len(trends))
	return trends, nil
}

func (h *HackerNewsMonitor) fetchTopStoryIDs(ctx context.Context) ([]int, error) {
	url := hnBaseURL + hnTopStories
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}

func (h *HackerNewsMonitor) fetchStory(ctx context.Context, id int) (*hnStory, error) {
	url := fmt.Sprintf(hnBaseURL+hnItem, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d for item %d", resp.StatusCode, id)
	}

	var story hnStory
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return nil, err
	}

	return &story, nil
}

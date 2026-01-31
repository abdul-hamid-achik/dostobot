package monitor

import (
	"context"
)

// Trend represents a detected trend from any source.
type Trend struct {
	Source      string
	ExternalID  string
	Title       string
	URL         string
	Description string
	Score       int
}

// Monitor is the interface for trend monitoring sources.
type Monitor interface {
	// Name returns the name of this monitor source.
	Name() string

	// FetchTrends retrieves current trends from the source.
	FetchTrends(ctx context.Context) ([]Trend, error)
}

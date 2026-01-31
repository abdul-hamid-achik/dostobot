package monitor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/abdulachik/dostobot/internal/db"
)

// Aggregator combines trends from multiple monitors.
type Aggregator struct {
	monitors []Monitor
	filter   *Filter
	store    *db.Store
}

// AggregatorConfig holds aggregator configuration.
type AggregatorConfig struct {
	Store    *db.Store
	Monitors []Monitor
	Filter   *Filter
}

// NewAggregator creates a new aggregator.
func NewAggregator(cfg AggregatorConfig) *Aggregator {
	filter := cfg.Filter
	if filter == nil {
		filter = NewFilter(FilterConfig{})
	}

	return &Aggregator{
		monitors: cfg.Monitors,
		filter:   filter,
		store:    cfg.Store,
	}
}

// FetchAndStore fetches trends from all monitors, filters them, and stores new ones.
func (a *Aggregator) FetchAndStore(ctx context.Context) ([]Trend, error) {
	var allTrends []Trend

	// Fetch from all monitors
	for _, monitor := range a.monitors {
		slog.Debug("fetching from monitor", "source", monitor.Name())

		trends, err := monitor.FetchTrends(ctx)
		if err != nil {
			slog.Error("monitor fetch failed",
				"source", monitor.Name(),
				"error", err,
			)
			continue
		}

		slog.Debug("fetched trends",
			"source", monitor.Name(),
			"count", len(trends),
		)

		allTrends = append(allTrends, trends...)
	}

	// Filter trends
	filtered := a.filter.FilterTrends(allTrends)
	slog.Debug("filtered trends",
		"before", len(allTrends),
		"after", len(filtered),
	)

	// Store new trends
	var newTrends []Trend
	for _, trend := range filtered {
		isNew, err := a.storeTrend(ctx, trend)
		if err != nil {
			slog.Error("failed to store trend",
				"title", trend.Title,
				"error", err,
			)
			continue
		}

		if isNew {
			newTrends = append(newTrends, trend)
		}
	}

	slog.Info("trend aggregation complete",
		"total_fetched", len(allTrends),
		"after_filter", len(filtered),
		"new_stored", len(newTrends),
	)

	return newTrends, nil
}

// storeTrend stores a trend if it's new, returns true if stored.
func (a *Aggregator) storeTrend(ctx context.Context, trend Trend) (bool, error) {
	// Check if trend already exists
	_, err := a.store.GetTrendBySourceAndExternalID(ctx, db.GetTrendBySourceAndExternalIDParams{
		Source:     trend.Source,
		ExternalID: sql.NullString{String: trend.ExternalID, Valid: true},
	})

	if err == nil {
		// Already exists
		return false, nil
	}

	if err != sql.ErrNoRows {
		return false, fmt.Errorf("check existing: %w", err)
	}

	// Store new trend
	_, err = a.store.CreateTrend(ctx, db.CreateTrendParams{
		Source:      trend.Source,
		ExternalID:  sql.NullString{String: trend.ExternalID, Valid: trend.ExternalID != ""},
		Title:       trend.Title,
		Url:         sql.NullString{String: trend.URL, Valid: trend.URL != ""},
		Description: sql.NullString{String: trend.Description, Valid: trend.Description != ""},
		Score:       sql.NullInt64{Int64: int64(trend.Score), Valid: true},
	})

	if err != nil {
		return false, fmt.Errorf("create trend: %w", err)
	}

	return true, nil
}

// GetUnmatchedTrends returns trends that haven't been matched to quotes yet.
func (a *Aggregator) GetUnmatchedTrends(ctx context.Context, limit int) ([]*db.Trend, error) {
	return a.store.ListUnmatchedTrends(ctx, int64(limit))
}

// HashTrend generates a unique hash for a trend (used for deduplication).
func HashTrend(trend Trend) string {
	data := fmt.Sprintf("%s:%s:%s", trend.Source, trend.ExternalID, trend.Title)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes
}

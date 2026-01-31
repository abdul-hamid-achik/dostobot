package monitor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMonitor is a mock implementation of Monitor for testing.
type mockMonitor struct {
	name   string
	trends []Trend
	err    error
}

func (m *mockMonitor) Name() string {
	return m.name
}

func (m *mockMonitor) FetchTrends(ctx context.Context) ([]Trend, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.trends, nil
}

func TestNewAggregator(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	agg := NewAggregator(AggregatorConfig{
		Store: store,
	})

	assert.NotNil(t, agg)
	assert.NotNil(t, agg.filter)
}

func TestAggregator_FetchAndStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	mock := &mockMonitor{
		name: "test",
		trends: []Trend{
			{Source: "test", ExternalID: "1", Title: "Test Trend 1", Score: 100},
			{Source: "test", ExternalID: "2", Title: "Test Trend 2", Score: 200},
		},
	}

	agg := NewAggregator(AggregatorConfig{
		Store:    store,
		Monitors: []Monitor{mock},
		Filter:   NewFilter(FilterConfig{}),
	})

	// First fetch should store all
	newTrends, err := agg.FetchAndStore(ctx)
	require.NoError(t, err)
	assert.Len(t, newTrends, 2)

	// Second fetch should find no new
	newTrends, err = agg.FetchAndStore(ctx)
	require.NoError(t, err)
	assert.Len(t, newTrends, 0)
}

func TestAggregator_FetchAndStore_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := db.NewStore(ctx, dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Migrate(ctx)
	require.NoError(t, err)

	mock := &mockMonitor{
		name: "test",
		trends: []Trend{
			{Source: "test", ExternalID: "1", Title: "Normal topic", Score: 100},
			{Source: "test", ExternalID: "2", Title: "Trump did something", Score: 200}, // Should be filtered
			{Source: "test", ExternalID: "3", Title: "Philosophy discussion", Score: 150},
		},
	}

	agg := NewAggregator(AggregatorConfig{
		Store:    store,
		Monitors: []Monitor{mock},
		Filter:   NewFilter(FilterConfig{}),
	})

	newTrends, err := agg.FetchAndStore(ctx)
	require.NoError(t, err)

	// Should have filtered out the Trump trend
	assert.Len(t, newTrends, 2)

	// Verify titles
	titles := make([]string, len(newTrends))
	for i, t := range newTrends {
		titles[i] = t.Title
	}
	assert.Contains(t, titles, "Normal topic")
	assert.Contains(t, titles, "Philosophy discussion")
	assert.NotContains(t, titles, "Trump did something")
}

func TestHashTrend(t *testing.T) {
	trend1 := Trend{Source: "test", ExternalID: "123", Title: "Test"}
	trend2 := Trend{Source: "test", ExternalID: "123", Title: "Test"}
	trend3 := Trend{Source: "test", ExternalID: "456", Title: "Test"}

	// Same trends should have same hash
	assert.Equal(t, HashTrend(trend1), HashTrend(trend2))

	// Different trends should have different hash
	assert.NotEqual(t, HashTrend(trend1), HashTrend(trend3))

	// Hash should be reasonable length
	assert.Len(t, HashTrend(trend1), 32) // 16 bytes = 32 hex chars
}

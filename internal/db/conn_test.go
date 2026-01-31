package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	t.Run("creates directory and database", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "subdir", "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Verify file exists
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)

		// Verify we can query
		var result int
		err = store.QueryRowContext(ctx, "SELECT 1").Scan(&result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("sets WAL mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		var mode string
		err = store.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode)
		assert.NoError(t, err)
		assert.Equal(t, "wal", mode)
	})

	t.Run("enables foreign keys", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		var fk int
		err = store.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk)
		assert.NoError(t, err)
		assert.Equal(t, 1, fk)
	})
}

func TestStore_Migrate(t *testing.T) {
	t.Run("applies migrations", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		err = store.Migrate(ctx)
		require.NoError(t, err)

		// Verify tables exist
		var tableName string
		err = store.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name='quotes'").Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "quotes", tableName)

		err = store.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "posts", tableName)

		err = store.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name='trends'").Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "trends", tableName)
	})

	t.Run("is idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		// Run twice
		err = store.Migrate(ctx)
		require.NoError(t, err)

		err = store.Migrate(ctx)
		require.NoError(t, err)

		// Still works
		count, err := store.CountQuotes(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("seeds config values", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		ctx := context.Background()
		store, err := NewStore(ctx, dbPath)
		require.NoError(t, err)
		defer store.Close()

		err = store.Migrate(ctx)
		require.NoError(t, err)

		// Check seeded config
		val, err := store.GetConfig(ctx, "monitor_interval")
		assert.NoError(t, err)
		assert.Equal(t, "30m", val)

		val, err = store.GetConfig(ctx, "max_posts_per_day")
		assert.NoError(t, err)
		assert.Equal(t, "6", val)
	})
}

func TestExtractUpMigration(t *testing.T) {
	t.Run("extracts up portion", func(t *testing.T) {
		content := `-- +migrate Up
CREATE TABLE test (id INTEGER);

-- +migrate Down
DROP TABLE test;
`
		result := extractUpMigration(content)
		assert.Equal(t, "CREATE TABLE test (id INTEGER);", result)
	})

	t.Run("handles no down marker", func(t *testing.T) {
		content := "CREATE TABLE test (id INTEGER);"
		result := extractUpMigration(content)
		assert.Equal(t, "CREATE TABLE test (id INTEGER);", result)
	})
}

// NewTestStore provides a test database for use in other packages.
func NewTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	store, err := NewStore(ctx, dbPath)
	require.NoError(t, err)

	err = store.Migrate(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

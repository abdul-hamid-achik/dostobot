package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abdulachik/dostobot/internal/db/migrations"
	_ "modernc.org/sqlite"
)

// Store wraps the database connection and provides access to queries.
type Store struct {
	*sql.DB
	*Queries
}

// NewStore creates a new database connection.
func NewStore(ctx context.Context, dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// Open connection
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection
	sqlDB.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well

	// Enable WAL mode and foreign keys
	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if _, err := sqlDB.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	store := &Store{
		DB:      sqlDB,
		Queries: New(sqlDB),
	}

	return store, nil
}

// Migrate runs all pending database migrations.
func (s *Store) Migrate(ctx context.Context) error {
	slog.Info("running database migrations")

	// Create migrations tracking table
	_, err := s.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Get applied migrations
	rows, err := s.QueryContext(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("scan migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate migrations: %w", err)
	}

	// Get migration files
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	// Apply pending migrations
	for _, file := range files {
		if applied[file] {
			slog.Debug("migration already applied", "file", file)
			continue
		}

		slog.Info("applying migration", "file", file)

		content, err := fs.ReadFile(migrations.FS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		// Extract up migration (before -- +migrate Down)
		sqlContent := extractUpMigration(string(content))

		// Execute migration in transaction
		tx, err := s.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, sqlContent); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", file, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", file); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", file, err)
		}

		slog.Info("migration applied successfully", "file", file)
	}

	return nil
}

// extractUpMigration extracts the "up" portion of a migration file.
func extractUpMigration(content string) string {
	// Find -- +migrate Down marker
	downMarker := "-- +migrate Down"
	idx := strings.Index(content, downMarker)
	if idx == -1 {
		return content
	}

	// Get content before Down marker
	up := content[:idx]

	// Remove -- +migrate Up marker if present
	upMarker := "-- +migrate Up"
	up = strings.TrimPrefix(up, upMarker)
	up = strings.TrimSpace(up)

	return up
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}

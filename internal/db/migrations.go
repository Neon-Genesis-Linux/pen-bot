package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ApplyMigrations brings the database schema up to date.
//
// Every migration in internal/db/migrations that the database has not run
// yet is applied, in the order their file names sort in. A migration that has
// already run is skipped, so this is safe to call on every start.
//
// A migration that fails leaves the schema as it was and the error says which
// file failed.
func ApplyMigrations(ctx context.Context, db *DB) error {
	if err := ensureSchemaMigrations(ctx, db); err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		if strings.HasPrefix(name, ".") {
			continue
		}
		applied, err := migrationApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		sqlText := strings.TrimSpace(string(content))
		if sqlText == "" {
			continue
		}

		if err := applyMigration(ctx, db, name, sqlText); err != nil {
			return err
		}
	}

	return nil
}

// applyMigration runs one migration file and records it as applied.
//
// The whole file is handed to the driver as a single statement, rather than
// being split on semicolons. The driver understands the quoting rules of the
// language, so a semicolon inside a string literal does not end the
// statement early, and the error it reports names the text that was wrong
// rather than a position in a list.
//
// The file runs in one transaction, so a failure leaves the schema as it was.
func applyMigration(ctx context.Context, db *DB, name, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		slog.Warn("migration failed", "migration", name, "error", err)
		return fmt.Errorf("migration %s failed: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES($1, $2)`, name, time.Now().UTC()); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s commit failed: %w", name, err)
	}

	return nil
}

func ensureSchemaMigrations(ctx context.Context, db *DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER))
	)
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func migrationApplied(ctx context.Context, db *DB, version string) (bool, error) {
	var existing string
	err := db.QueryRowContext(ctx, `SELECT version FROM schema_migrations WHERE version = $1`, version).Scan(&existing)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

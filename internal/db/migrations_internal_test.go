package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestApplyMigrationHandlesSemicolonInLiteral checks that a migration
// containing a semicolon inside a string literal is applied whole.
//
// The file is handed to the driver as one statement rather than being split
// on semicolons, so a semicolon in a literal does not cut the statement in
// half. This is the case a split would get wrong.
func TestApplyMigrationHandlesSemicolonInLiteral(t *testing.T) {
	database, err := New(Config{Path: filepath.Join(t.TempDir(), "pen-bot.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	if err := ensureSchemaMigrations(ctx, database); err != nil {
		t.Fatalf("ensure schema migrations: %v", err)
	}

	const migration = `
CREATE TABLE notes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	body TEXT NOT NULL DEFAULT 'first; second; third'
);
INSERT INTO notes (body) VALUES ('a; b');
INSERT INTO notes DEFAULT VALUES;
`

	if err := applyMigration(ctx, database, "001_notes.sql", migration); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	var body string
	if err := database.QueryRowContext(ctx, `SELECT body FROM notes WHERE id = 1`).Scan(&body); err != nil {
		t.Fatalf("select: %v", err)
	}
	if body != "a; b" {
		t.Errorf("got body %q, want %q", body, "a; b")
	}

	var defaultBody string
	if err := database.QueryRowContext(ctx, `SELECT body FROM notes WHERE id = 2`).Scan(&defaultBody); err != nil {
		t.Fatalf("select default: %v", err)
	}
	if defaultBody != "first; second; third" {
		t.Errorf("got default %q, want %q", defaultBody, "first; second; third")
	}
}

// TestApplyMigrationRollsBackOnFailure checks that a migration that fails
// leaves the schema as it was.
//
// The whole file runs in one transaction, so a statement that fails undoes
// the statements before it.
func TestApplyMigrationRollsBackOnFailure(t *testing.T) {
	database, err := New(Config{Path: filepath.Join(t.TempDir(), "pen-bot.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	const migration = `
CREATE TABLE before_failure (id INTEGER PRIMARY KEY);
CREATE TABLE after_failure (id INTEGER PRIMARY KEY, broken INTEGER NOT NULL);
CREATE TABLE never_reached (id INTEGER PRIMARY KEY);
`

	if err := applyMigration(ctx, database, "001_broken.sql", migration); err == nil {
		t.Fatal("got nil error, want the migration to fail")
	}

	var count int
	if err := database.QueryRowContext(ctx, `
		SELECT count(*) FROM sqlite_master
		WHERE type = 'table' AND name IN ('before_failure', 'after_failure', 'never_reached')
	`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d tables left behind, want 0", count)
	}
}

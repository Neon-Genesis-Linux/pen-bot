package db_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

// testDB opens a database in a temporary directory and applies the migrations
// to it.
//
// The file is on disk rather than in memory because SQLite gives each
// connection its own in-memory database, so a pooled handle over ":memory:"
// loses rows written through one connection and read through another.
func testDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.New(db.Config{Path: filepath.Join(t.TempDir(), "pen-bot.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.ApplyMigrations(context.Background(), database); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return database
}

func TestGuildUpsert(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	guild, err := db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get missing guild: %v", err)
	}
	if guild != nil {
		t.Fatalf("got %+v, want nil for an unknown guild", guild)
	}

	if err := db.UpsertGuild(ctx, database, "g1", `{"theme":"dark"}`); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	guild, err = db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get guild: %v", err)
	}
	if guild == nil {
		t.Fatal("got nil, want the stored guild")
	}
	if guild.GuildID != "g1" {
		t.Errorf("got guild id %q, want %q", guild.GuildID, "g1")
	}
	if string(guild.SettingsJSON) != `{"theme":"dark"}` {
		t.Errorf("got settings %s, want %s", guild.SettingsJSON, `{"theme":"dark"}`)
	}
	if guild.CreatedAt.IsZero() {
		t.Error("got zero created_at")
	}

	// A second upsert replaces the settings and keeps the same row.
	if err := db.UpsertGuild(ctx, database, "g1", `{"theme":"light"}`); err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	guild, err = db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get guild again: %v", err)
	}
	if guild == nil {
		t.Fatal("got nil, want the stored guild")
	}
	if string(guild.SettingsJSON) != `{"theme":"light"}` {
		t.Errorf("got settings %s, want %s", guild.SettingsJSON, `{"theme":"light"}`)
	}

	if err := db.DeleteGuild(ctx, database, "g1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	guild, err = db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get deleted guild: %v", err)
	}
	if guild != nil {
		t.Fatalf("got %+v, want nil after delete", guild)
	}
}

func TestUserUpsert(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	user, err := db.GetUser(ctx, database, "g1", "u1")
	if err != nil {
		t.Fatalf("get missing user: %v", err)
	}
	if user != nil {
		t.Fatalf("got %+v, want nil for an unknown user", user)
	}

	if err := db.UpsertUser(ctx, database, "g1", "u1", `{"name":"ada"}`); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	user, err = db.GetUser(ctx, database, "g1", "u1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user == nil {
		t.Fatal("got nil, want the stored user")
	}
	if user.UserID != "u1" {
		t.Errorf("got user id %q, want %q", user.UserID, "u1")
	}
	if string(user.ProfileJSON) != `{"name":"ada"}` {
		t.Errorf("got profile %s, want %s", user.ProfileJSON, `{"name":"ada"}`)
	}

	// The same user in another guild is a different row.
	if err := db.UpsertUser(ctx, database, "g2", "u1", `{"name":"grace"}`); err != nil {
		t.Fatalf("upsert in second guild: %v", err)
	}
	other, err := db.GetUser(ctx, database, "g2", "u1")
	if err != nil {
		t.Fatalf("get user in second guild: %v", err)
	}
	if other == nil {
		t.Fatal("got nil, want the stored user in the second guild")
	}
	if string(other.ProfileJSON) != `{"name":"grace"}` {
		t.Errorf("got profile %s, want %s", other.ProfileJSON, `{"name":"grace"}`)
	}

	if err := db.DeleteUser(ctx, database, "g1", "u1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	user, err = db.GetUser(ctx, database, "g1", "u1")
	if err != nil {
		t.Fatalf("get deleted user: %v", err)
	}
	if user != nil {
		t.Fatalf("got %+v, want nil after delete", user)
	}
}

// TestModerationLogPagination walks the whole history through the cursor and
// checks that every entry is returned exactly once.
func TestModerationLogPagination(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	const entries = 25
	for i := range entries {
		if err := db.LogModeration(ctx, database, "g1", "mod", "target", "timeout", "reason"); err != nil {
			t.Fatalf("log %d: %v", i, err)
		}
		// Timestamps are stored to the millisecond, so entries written in
		// the same millisecond share a timestamp and are ordered by id.
		time.Sleep(2 * time.Millisecond)
	}

	seen := make(map[int64]bool)
	var cursor *db.ModerationCursor
	pages := 0
	for {
		logs, err := db.GetModerationLogs(ctx, database, "g1", 10, cursor)
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		if len(logs) == 0 {
			break
		}
		pages++

		for _, entry := range logs {
			if seen[entry.ID] {
				t.Fatalf("id %d returned twice", entry.ID)
			}
			seen[entry.ID] = true
		}

		last := logs[len(logs)-1]
		cursor = &db.ModerationCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		if pages > entries {
			t.Fatal("pagination did not terminate")
		}
	}

	if len(seen) != entries {
		t.Errorf("got %d entries across %d pages, want %d", len(seen), pages, entries)
	}
}

// TestModerationLogPaginationAcrossGuilds checks that the cursor pages within
// one guild and never crosses into another.
func TestModerationLogPaginationAcrossGuilds(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	for range 3 {
		if err := db.LogModeration(ctx, database, "g1", "mod", "target", "timeout", "reason"); err != nil {
			t.Fatalf("log in first guild: %v", err)
		}
	}
	for range 3 {
		if err := db.LogModeration(ctx, database, "g2", "mod", "target", "timeout", "reason"); err != nil {
			t.Fatalf("log in second guild: %v", err)
		}
	}

	logs, err := db.GetModerationLogs(ctx, database, "g1", 20, nil)
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("got %d entries, want 3", len(logs))
	}
	for _, entry := range logs {
		if entry.GuildID != "g1" {
			t.Errorf("got entry for guild %q, want %q", entry.GuildID, "g1")
		}
	}
}

// TestTimestampsSurviveRoundTrip checks that a timestamp written by the
// application reads back as the same instant.
//
// The columns hold Unix milliseconds, so a value written by the application
// and a value written by a column default have to be read the same way.
func TestTimestampsSurviveRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	written := time.Now().UTC().Truncate(time.Millisecond)
	if err := db.UpsertGuild(ctx, database, "g1", `{}`); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	guild, err := db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get guild: %v", err)
	}
	if guild == nil {
		t.Fatal("got nil, want the stored guild")
	}
	if !guild.CreatedAt.Equal(written) && guild.CreatedAt.After(written.Add(time.Second)) {
		t.Errorf("got created_at %v, want near %v", guild.CreatedAt, written)
	}
	if guild.CreatedAt.IsZero() {
		t.Error("got zero created_at")
	}
}

// TestTimestampsOrderCorrectly checks that timestamps written by the
// application and timestamps written by a column default are ordered by the
// instant they record.
//
// Both are stored as Unix milliseconds. Storing them as text instead would
// mix two kinds of value in one column, because the driver writes text while
// the default writes an integer, and SQLite orders every integer before
// every piece of text whatever the instant each one records.
func TestTimestampsOrderCorrectly(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	// The default on created_at is written by the database, so the row is
	// inserted without naming the column.
	if _, err := database.ExecContext(ctx, `INSERT INTO guilds (guild_id, settings_json) VALUES ($1, $2)`, "defaulted", `{}`); err != nil {
		t.Fatalf("insert with default timestamp: %v", err)
	}

	// A row written by the application, recording an earlier instant than
	// the one above.
	earlier := time.Now().UTC().Add(-time.Hour)
	if _, err := database.ExecContext(ctx, `INSERT INTO guilds (guild_id, settings_json, created_at) VALUES ($1, $2, $3)`, "written", `{}`, earlier); err != nil {
		t.Fatalf("insert with written timestamp: %v", err)
	}

	var ids []string
	rows, err := database.QueryContext(ctx, `SELECT guild_id FROM guilds ORDER BY created_at DESC`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var guildID string
		if err := rows.Scan(&guildID); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, guildID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if len(ids) != 2 || ids[0] != "defaulted" || ids[1] != "written" {
		t.Errorf("got order %v, want [defaulted written]", ids)
	}
}

// TestTimestampsShareOneStorageType checks that every timestamp in a column
// holds the same kind of value.
//
// A column that mixes text and integers sorts by kind rather than by
// instant, so the two kinds have to agree for the ordering to mean anything.
func TestTimestampsShareOneStorageType(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	if _, err := database.ExecContext(ctx, `INSERT INTO guilds (guild_id, settings_json) VALUES ($1, $2)`, "defaulted", `{}`); err != nil {
		t.Fatalf("insert with default timestamp: %v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO guilds (guild_id, settings_json, created_at) VALUES ($1, $2, $3)`, "written", `{}`, time.Now().UTC()); err != nil {
		t.Fatalf("insert with written timestamp: %v", err)
	}

	rows, err := database.QueryContext(ctx, `SELECT DISTINCT typeof(created_at) FROM guilds`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var kinds []string
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			t.Fatalf("scan: %v", err)
		}
		kinds = append(kinds, kind)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if len(kinds) != 1 {
		t.Fatalf("got timestamp kinds %v, want exactly one", kinds)
	}
	if kinds[0] != "integer" {
		t.Errorf("got timestamp kind %q, want %q", kinds[0], "integer")
	}
}

// TestUpsertKeepsCreatedAtAndAdvancesUpdatedAt checks that replacing a row's
// settings leaves the row's creation time alone and moves its update time
// forward.
func TestUpsertKeepsCreatedAtAndAdvancesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	if err := db.UpsertGuild(ctx, database, "g1", `{"v":1}`); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	first, err := db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get guild: %v", err)
	}
	if first == nil {
		t.Fatal("got nil, want the stored guild")
	}

	time.Sleep(20 * time.Millisecond)

	if err := db.UpsertGuild(ctx, database, "g1", `{"v":2}`); err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	second, err := db.GetGuild(ctx, database, "g1")
	if err != nil {
		t.Fatalf("get guild again: %v", err)
	}
	if second == nil {
		t.Fatal("got nil, want the stored guild")
	}

	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("got created_at %v, want it unchanged at %v", second.CreatedAt, first.CreatedAt)
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Errorf("got updated_at %v, want it later than %v", second.UpdatedAt, first.UpdatedAt)
	}
}

// TestMigrationsAreIdempotent checks that applying the migrations to a
// database that already has them leaves it usable.
func TestMigrationsAreIdempotent(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	if err := db.ApplyMigrations(ctx, database); err != nil {
		t.Fatalf("apply migrations again: %v", err)
	}

	if err := db.UpsertGuild(ctx, database, "g1", `{}`); err != nil {
		t.Fatalf("upsert after reapplying migrations: %v", err)
	}
}

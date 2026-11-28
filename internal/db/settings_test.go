package db_test

import (
	"context"
	"testing"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

// roleNineAdmin is the guild document with a single admin role, compared often
// enough that goconst wants it pulled out of the assertions.
const roleNineAdmin = `{"roles":{"9":"admin"}}`

// These exercise the settings SQL against the real SQLite driver, so the
// SQLite JSON and date functions these queries rely on are pinned here. They
// are in package db_test, which is the only package that has tests and needs
// no server.

func TestSetGuildDocumentRoundTripsAndUpserts(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)
	const guild = "g1"

	if raw, err := db.GetGuildDocument(ctx, database, guild); err != nil || raw != nil {
		t.Fatalf("missing guild document = %q, %v: want nil", raw, err)
	}

	if err := db.SetGuildDocument(ctx, database, guild, `{"roles":{}}`); err != nil {
		t.Fatalf("set: %v", err)
	}
	raw, err := db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != `{"roles":{}}` {
		t.Fatalf("round trip = %q, %v: want the stored document", raw, err)
	}

	if err := db.SetGuildDocument(ctx, database, guild, roleNineAdmin); err != nil {
		t.Fatalf("upsert set: %v", err)
	}
	raw, err = db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != roleNineAdmin {
		t.Fatalf("after upsert = %q, %v: want the replaced document", raw, err)
	}
}

func TestSetGuildRoleTierCreatesThenAdds(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)
	const guild = "g1"

	if err := db.SetGuildRoleTier(ctx, database, guild, "9", "admin"); err != nil {
		t.Fatalf("set first role: %v", err)
	}
	raw, err := db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != roleNineAdmin {
		t.Fatalf("after first role = %q, %v", raw, err)
	}

	// A second, different role must not overwrite the first: the path is written
	// individually rather than by replacing the whole document.
	if err := db.SetGuildRoleTier(ctx, database, guild, "10", "moderator"); err != nil {
		t.Fatalf("set second role: %v", err)
	}
	raw, err = db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != `{"roles":{"9":"admin","10":"moderator"}}` {
		t.Fatalf("after second role = %q, %v: want both roles", raw, err)
	}
}

func TestRemoveGuildRoleDropsTheRoleAndReportsIt(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)
	const guild = "g1"

	if err := db.SetGuildRoleTier(ctx, database, guild, "9", "admin"); err != nil {
		t.Fatalf("set role: %v", err)
	}
	if err := db.SetGuildRoleTier(ctx, database, guild, "10", "moderator"); err != nil {
		t.Fatalf("set role: %v", err)
	}

	removed, err := db.RemoveGuildRole(ctx, database, guild, "10")
	if err != nil || !removed {
		t.Fatalf("remove = %v, %v: want true", removed, err)
	}
	raw, err := db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != roleNineAdmin {
		t.Fatalf("after remove = %q, %v: want only the remaining role", raw, err)
	}

	// Removing a role the guild never configured touches no row.
	if removed, err := db.RemoveGuildRole(ctx, database, guild, "11"); err != nil || removed {
		t.Fatalf("remove unknown = %v, %v: want false", removed, err)
	}
	raw, err = db.GetGuildDocument(ctx, database, guild)
	if err != nil || string(raw) != roleNineAdmin {
		t.Fatalf("document changed on an unguarded path: %q, %v", raw, err)
	}
}

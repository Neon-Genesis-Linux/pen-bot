package db_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
)

// TestModerationLogQueryPlan checks that the pagination query reads the
// moderation log in index order rather than sorting each page again.
//
// The cursor compares the pair (created_at, id), so the index has to carry
// both columns to break the ties between entries written in the same
// millisecond. When it does not, SQLite sorts the page with a temporary
// B-tree over the whole table, which grows with the table.
//
// The plan is asserted rather than the timing, because a plan is what decides
// the cost, and it holds however fast the machine is.
func TestModerationLogQueryPlan(t *testing.T) {
	ctx := context.Background()
	database := testDB(t)

	for range 50 {
		if err := db.LogModeration(ctx, database, "g1", "mod", "target", "timeout", "reason"); err != nil {
			t.Fatalf("log: %v", err)
		}
	}

	rows, err := database.QueryContext(ctx, `
		EXPLAIN QUERY PLAN
		SELECT id, guild_id, moderator_id, target_id, action, reason, created_at
		FROM moderation_logs
		WHERE guild_id = $1 AND (created_at, id) < ($2, $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, "g1", time.Now().UTC().Add(time.Hour), int64(1)<<62, 20)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var details []string
	for rows.Next() {
		var id, parent, notUsed int64
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	plan := strings.Join(details, " | ")
	if strings.Contains(plan, "TEMP B-TREE") {
		t.Errorf("got plan %q, want no temporary sort", plan)
	}
	if !strings.Contains(plan, "idx_moderation_logs_guild_created") {
		t.Errorf("got plan %q, want it to use idx_moderation_logs_guild_created", plan)
	}
}

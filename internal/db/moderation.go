package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// LogModeration records a moderation action.
func LogModeration(ctx context.Context, db *DB, guildID, moderatorID, targetID, action, reason string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	_, err := db.ExecContext(ctx, `
	INSERT INTO moderation_logs (guild_id, moderator_id, target_id, action, reason, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`, guildID, moderatorID, targetID, action, reason, time.Now().UTC())
	return err
}

// GetModerationLogs returns moderation logs for a guild, newest first.
//
// A cursor pages through the history. The cursor names the position to
// continue from, and the query returns the entries strictly after it, so a
// caller walking the pages sees every entry exactly once even while new
// entries are being written.
func GetModerationLogs(ctx context.Context, db *DB, guildID string, limit int, cursor *ModerationCursor) ([]*ModerationLog, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guildID required")
	}
	if limit <= 0 {
		limit = 20
	}

	var args []interface{}
	var clauses []string

	args = append(args, guildID)
	clauses = append(clauses, fmt.Sprintf("guild_id = $%d", len(args)))

	if cursor != nil {
		args = append(args, cursor.CreatedAt, cursor.ID)
		clauses = append(clauses, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}

	args = append(args, limit)
	query := fmt.Sprintf(`
	SELECT id, guild_id, moderator_id, target_id, action, reason, created_at
	FROM moderation_logs
	WHERE %s
	ORDER BY created_at DESC, id DESC
	LIMIT $%d`, strings.Join(clauses, " AND "), len(args))

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	logs := make([]*ModerationLog, 0)
	for rows.Next() {
		var logEntry ModerationLog
		if err := rows.Scan(
			&logEntry.ID,
			&logEntry.GuildID,
			&logEntry.ModeratorID,
			&logEntry.TargetID,
			&logEntry.Action,
			&logEntry.Reason,
			&logEntry.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

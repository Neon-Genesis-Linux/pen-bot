package db

import (
	"context"
	"database/sql"
	"fmt"
)

func GetUser(ctx context.Context, db *DB, guildID, userID string) (*User, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guildID required")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID required")
	}
	var user User
	err := db.QueryRowContext(ctx, `
	SELECT id, guild_id, user_id, profile_json, created_at, updated_at
	FROM users
	WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(
		&user.ID,
		&user.GuildID,
		&user.UserID,
		&user.ProfileJSON,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpsertUser(ctx context.Context, db *DB, guildID, userID, profileJSON string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	if userID == "" {
		return fmt.Errorf("userID required")
	}
	_, err := db.ExecContext(ctx, `
	INSERT INTO users (guild_id, user_id, profile_json, created_at, updated_at)
	VALUES ($1, $2, $3::jsonb, now(), now())
	ON CONFLICT (guild_id, user_id) DO UPDATE SET
		profile_json = EXCLUDED.profile_json,
		updated_at = now()
	`, guildID, userID, profileJSON)
	return err
}

func DeleteUser(ctx context.Context, db *DB, guildID, userID string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	if userID == "" {
		return fmt.Errorf("userID required")
	}
	_, err := db.ExecContext(ctx, `DELETE FROM users WHERE guild_id = $1 AND user_id = $2`, guildID, userID)
	return err
}

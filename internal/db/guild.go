package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GetGuild returns the stored settings for a guild, or nil when the guild has
// no row yet.
func GetGuild(ctx context.Context, db *DB, guildID string) (*Guild, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guildID required")
	}
	var guild Guild
	err := db.QueryRowContext(ctx, `
	SELECT id, guild_id, settings_json, created_at, updated_at
	FROM guilds
	WHERE guild_id = $1
	`, guildID).Scan(
		&guild.ID,
		&guild.GuildID,
		&guild.SettingsJSON,
		&guild.CreatedAt,
		&guild.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &guild, nil
}

// UpsertGuild stores the settings for a guild, replacing any that are already
// there.
func UpsertGuild(ctx context.Context, db *DB, guildID string, settingsJSON string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
	INSERT INTO guilds (guild_id, settings_json, created_at, updated_at)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (guild_id) DO UPDATE SET
		settings_json = excluded.settings_json,
		updated_at = $4
	`, guildID, settingsJSON, now, now)
	return err
}

// DeleteGuild removes the stored settings for a guild.
func DeleteGuild(ctx context.Context, db *DB, guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	_, err := db.ExecContext(ctx, `DELETE FROM guilds WHERE guild_id = $1`, guildID)
	return err
}

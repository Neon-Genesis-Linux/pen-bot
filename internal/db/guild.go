package db

import (
	"context"
	"database/sql"
	"fmt"
)

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

func UpsertGuild(ctx context.Context, db *DB, guildID string, settingsJSON string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	_, err := db.ExecContext(ctx, `
	INSERT INTO guilds (guild_id, settings_json, created_at, updated_at)
	VALUES ($1, $2::jsonb, now(), now())
	ON CONFLICT (guild_id) DO UPDATE SET
		settings_json = EXCLUDED.settings_json,
		updated_at = now()
	`, guildID, settingsJSON)
	return err
}

func DeleteGuild(ctx context.Context, db *DB, guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guildID required")
	}
	_, err := db.ExecContext(ctx, `DELETE FROM guilds WHERE guild_id = $1`, guildID)
	return err
}

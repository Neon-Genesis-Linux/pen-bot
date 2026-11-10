// Package db stores the bot's data in a SQLite database.
//
// The data is small and belongs to a single guild, so it lives in a file
// rather than in a database server. The package opens that file, applies the
// migrations that bring its schema up to date, and offers the queries the
// bot runs against it.
//
// Timestamps are stored as Unix milliseconds in INTEGER columns. The driver
// reads and writes that representation, so a caller passes and receives
// time.Time and never handles the integer form.
package db

import "time"

// Guild holds the settings stored for a guild.
type Guild struct {
	ID           int64
	GuildID      string
	SettingsJSON []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// User holds the profile stored for a user in a guild.
//
// A user is identified by the guild they are in as well as by their own id,
// so the same user in two guilds is two rows.
type User struct {
	ID          int64
	GuildID     string
	UserID      string
	ProfileJSON []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ModerationLog holds one recorded moderation action.
type ModerationLog struct {
	ID          int64
	GuildID     string
	ModeratorID string
	TargetID    string
	Action      string
	Reason      string
	CreatedAt   time.Time
}

// ModerationCursor names a position in the moderation log.
//
// It is the pair of the timestamp and the id of the last entry a caller has
// seen, and it is used to ask for the entries that come before it.
type ModerationCursor struct {
	CreatedAt time.Time
	ID        int64
}

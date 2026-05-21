package db

import "time"

type Guild struct {
	ID           int64
	GuildID      string
	SettingsJSON []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type User struct {
	ID          int64
	GuildID     string
	UserID      string
	ProfileJSON []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ModerationLog struct {
	ID          int64
	GuildID     string
	ModeratorID string
	TargetID    string
	Action      string
	Reason      string
	CreatedAt   time.Time
}

type ModerationCursor struct {
	CreatedAt time.Time
	ID        int64
}

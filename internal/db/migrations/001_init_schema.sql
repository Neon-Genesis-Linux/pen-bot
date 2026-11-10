CREATE TABLE IF NOT EXISTS guilds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL UNIQUE,
    settings_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(settings_json)),
    created_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER)),
    updated_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER))
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    profile_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(profile_json)),
    created_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER)),
    updated_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER)),
    UNIQUE (guild_id, user_id)
);

CREATE TABLE IF NOT EXISTS moderation_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    moderator_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER))
);

-- The moderation log is read newest first within a guild, so the index carries
-- the sort order. It ends with id because the pagination cursor compares the
-- pair (created_at, id): an index without id leaves SQLite sorting each page
-- again to break the ties, which it does with a temporary B-tree over the
-- whole table.
CREATE INDEX IF NOT EXISTS idx_moderation_logs_guild_created ON moderation_logs (guild_id, created_at DESC, id DESC);

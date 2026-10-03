package core

import "github.com/disgoorg/snowflake/v2"

// IsGuild reports whether the interaction that produced e originated in a
// guild. GuildID is nil for direct messages, where guild-scoped commands and
// REST calls (member updates, moderative actions) are unavailable.
func IsGuild[T interface{ GuildID() *snowflake.ID }](e T) bool {
	return e.GuildID() != nil
}

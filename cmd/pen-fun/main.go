package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/disgoorg/snowflake/v2"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/community"
	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
	_ "github.com/Neon-Genesis-Linux/pen-bot/internal/logger"
)

func main() {
	community.Register()

	guildID := snowflake.GetEnv("GUILD_ID")
	var guildIDs []snowflake.ID
	if guildID != 0 {
		guildIDs = append(guildIDs, guildID)
	}

	if err := core.Start(context.Background(), os.Getenv("BOT_TOKEN"), guildIDs); err != nil {
		slog.Error("failed to start pen-fun bot", slog.Any("error", err))
	}
}

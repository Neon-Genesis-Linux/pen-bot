package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/community"
	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
	"github.com/disgoorg/disgo/events"
)

func main() {
	community.Register()
	if err := core.Start(context.Background(), os.Getenv("BOT_TOKEN"), onMessageCreate); err != nil {
		slog.Error("failed to start pen-fun bot", slog.Any("error", err))
	}
}

func onMessageCreate(event *events.MessageCreate) {
	core.DispatchCommand(event)
}

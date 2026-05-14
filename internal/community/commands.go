package community

import (
	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// Register registers community commands
func Register() {
	core.RegisterCommand("ping", handlePing)
	core.RegisterCommand("pong", handlePong)
}

func handlePing(event *events.MessageCreate) {
	_, _ = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent("pong"))
}

func handlePong(event *events.MessageCreate) {
	_, _ = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent("ping"))
}

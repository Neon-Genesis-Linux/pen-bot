package community

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

// Register registers community commands
func Register() {
	core.RegisterCommands(
		discord.SlashCommandCreate{
			Name:        "ping",
			Description: "Respond with Pong!",
		},
		discord.SlashCommandCreate{
			Name:        "pong",
			Description: "Respond with Ping!",
		},
	)
	registerXkcdCommands()

	h := core.Mux()
	h.SlashCommand("/ping", handlePing)
	h.SlashCommand("/pong", handlePong)
}

func handlePing(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	return e.CreateMessage(discord.MessageCreate{Content: "pong"})
}

func handlePong(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	return e.CreateMessage(discord.MessageCreate{Content: "ping"})
}

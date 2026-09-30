package community

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

// Register adds the community slash commands to the bot's command list and
// routes their handlers. Call it before Start: a command is only synced to
// Discord if its definition is registered here, and only answered if a handler
// is routed for it.
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
	registerTldrCommands()

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

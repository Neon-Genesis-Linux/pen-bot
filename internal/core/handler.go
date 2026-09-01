package core

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
)

var (
	router          = handler.New()
	commandDefs     []discord.ApplicationCommandCreate
	requiredIntents gateway.Intents
)

// Mux returns the global interaction router used to register command handlers.
func Mux() *handler.Mux {
	return router
}

// RegisterCommands appends application command definitions to be synced with Discord.
func RegisterCommands(cmds ...discord.ApplicationCommandCreate) {
	commandDefs = append(commandDefs, cmds...)
}

// RegisterIntents ORs the given gateway intents into the set requested when
// connecting to Discord. Modules that need intent-gated gateway events register
// their intents here so the bot never requests intents it does not use (e.g.
// the privileged MESSAGE_CONTENT intent).
func RegisterIntents(intents ...gateway.Intents) {
	requiredIntents = requiredIntents.Add(intents...)
}

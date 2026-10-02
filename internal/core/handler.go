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

// RegisterCommands appends application command definitions to the set Start
// syncs. A definition with no handler routed for it reaches Discord and does
// nothing.
func RegisterCommands(cmds ...discord.ApplicationCommandCreate) {
	commandDefs = append(commandDefs, cmds...)
}

// RegisterIntents adds the given gateway intents to the set Start requests
// when identifying with Discord. Adding is cumulative, so two calls accumulate
// rather than replace. The zero value requests none, which leaves the guild
// cache empty and stops message events, though interaction events are not gated
// on intents and arrive either way.
func RegisterIntents(intents ...gateway.Intents) {
	requiredIntents = requiredIntents.Add(intents...)
}

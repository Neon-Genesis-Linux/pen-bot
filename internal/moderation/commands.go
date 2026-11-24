// Package moderation provides the bot's moderation slash commands.
//
// Call Register before core.Start so the command definitions are synced and
// the handlers are routed when the gateway opens.
package moderation

// Register registers the moderation commands.
func Register() {
	// Not moderation per se, but as noted in issue #26, it must be in here for permission reasons
	registerSelfTimeoutCommand()
}

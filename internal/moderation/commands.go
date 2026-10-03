package moderation

// Register registers moderation commands with the bot
func Register() {
	// Not moderation per se, but as noted in issue #26, it must be in here for permission reasons
	registerSelfTimeoutCommand()
}

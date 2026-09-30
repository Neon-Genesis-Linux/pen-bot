package moderation

// Register registers the moderation commands.
func Register() {
	// Not moderation per se, but as noted in issue #26, it must be in here for permission reasons
	registerSelfTimeoutCommand()
}

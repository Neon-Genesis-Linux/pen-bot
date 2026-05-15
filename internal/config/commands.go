package config

import (
	"os"
	"strings"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
	"github.com/Neon-Genesis-Linux/pen-bot/internal/messaging"
	"github.com/disgoorg/disgo/events"
)

const ownerEnv = "BOT_OWNER_ID"

// Register registers configuration commands with the bot.
func Register() {
	core.RegisterCommandPath([]string{"config", "prefix", "set"}, handlePrefixSet)
}

func handlePrefixSet(event *events.MessageCreate, args []string) {
	if !isOwner(event) {
		_ = messaging.SendReply(event, "Unauthorized: only bot owner can run config commands.")
		return
	}

	if len(args) < 1 {
		_ = messaging.SendReply(event, "Usage: "+core.GetBotPrefix()+"config prefix set <newprefix>")
		return
	}

	newPrefix := strings.TrimSpace(args[0])
	if newPrefix == "" {
		_ = messaging.SendReply(event, "Prefix cannot be empty.")
		return
	}

	core.SetBotPrefix(newPrefix)
	_ = messaging.Send(event, "Command prefix set to `"+newPrefix+"`.")
}

func isOwner(event *events.MessageCreate) bool {
	ownerID := os.Getenv(ownerEnv)
	if ownerID == "" {
		return false
	}
	return event.Message.Author.ID.String() == ownerID
}

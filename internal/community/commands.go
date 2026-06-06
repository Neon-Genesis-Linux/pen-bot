package community

import (
	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
	"github.com/Neon-Genesis-Linux/pen-bot/internal/messaging"
	"github.com/disgoorg/disgo/events"
)

// Register registers community commands
func Register() {
	core.RegisterSimpleCommand("ping", handlePing)
	core.RegisterSimpleCommand("pong", handlePong)
}

func handlePing(event *events.MessageCreate) {
	_ = messaging.SendReply(event, "pong")
}

func handlePong(event *events.MessageCreate) {
	_ = messaging.SendReply(event, "ping")
}

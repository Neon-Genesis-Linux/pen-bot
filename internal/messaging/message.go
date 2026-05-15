package messaging

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// SendOptions configures message sending behavior
type SendOptions struct {
	Reply bool // If true, reply to the triggering message
}

// SendMessage sends a message with optional reply.
// Errors are logged but not returned to user.
// Returns error for caller's information if needed.
func SendMessage(event *events.MessageCreate, content string, opts *SendOptions) error {
	if opts == nil {
		opts = &SendOptions{}
	}

	builder := discord.NewMessageCreate().WithContent(content)

	if opts.Reply {
		messageID := event.Message.ID
		builder = builder.WithMessageReference(&discord.MessageReference{
			MessageID: &messageID,
		})
	}

	_, err := event.Client().Rest.CreateMessage(event.ChannelID, builder)
	if err != nil {
		slog.Error("failed to send message",
			slog.String("channel_id", event.ChannelID.String()),
			slog.String("content", content),
			slog.Any("error", err),
		)
	}
	return err
}

// Send sends a simple message (no reply)
func Send(event *events.MessageCreate, content string) error {
	return SendMessage(event, content, nil)
}

// SendReply sends a message as a reply to the user's message
func SendReply(event *events.MessageCreate, content string) error {
	return SendMessage(event, content, &SendOptions{Reply: true})
}

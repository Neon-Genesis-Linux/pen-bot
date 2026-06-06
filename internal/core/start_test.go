package core

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func TestDispatchExactMatch(t *testing.T) {
	called := false
	RegisterSimpleCommand("test-exact", func(e *events.MessageCreate) { called = true })

	DispatchCommand(&events.MessageCreate{
		GenericMessage: &events.GenericMessage{
			Message: discord.Message{
				Content: "!test-exact",
				Author:  discord.User{Bot: false},
			},
		},
	})

	if !called {
		t.Fatal("expected handler called")
	}
}

func TestDispatchIgnoresBot(t *testing.T) {
	called := false
	RegisterSimpleCommand("test-bot", func(e *events.MessageCreate) { called = true })

	DispatchCommand(&events.MessageCreate{
		GenericMessage: &events.GenericMessage{
			Message: discord.Message{
				Content: "!test-bot",
				Author:  discord.User{Bot: true},
			},
		},
	})

	if called {
		t.Fatal("expected bot message ignored")
	}
}

func TestDispatchNoPrefix(t *testing.T) {
	called := false
	RegisterSimpleCommand("test-noprefix", func(e *events.MessageCreate) { called = true })

	DispatchCommand(&events.MessageCreate{
		GenericMessage: &events.GenericMessage{
			Message: discord.Message{
				Content: "test-noprefix",
				Author:  discord.User{Bot: false},
			},
		},
	})

	if called {
		t.Fatal("expected no dispatch without prefix")
	}
}

func TestDispatchUnknownCommand(t *testing.T) {
	DispatchCommand(&events.MessageCreate{
		GenericMessage: &events.GenericMessage{
			Message: discord.Message{
				Content: "!unknown-cmd",
				Author:  discord.User{Bot: false},
			},
		},
	})
}

func TestCustomPrefix(t *testing.T) {
	oldPrefix := botPrefix
	botPrefix = "?"
	t.Cleanup(func() { botPrefix = oldPrefix })

	called := false
	RegisterSimpleCommand("test-custom", func(e *events.MessageCreate) { called = true })

	DispatchCommand(&events.MessageCreate{
		GenericMessage: &events.GenericMessage{
			Message: discord.Message{
				Content: "?test-custom",
				Author:  discord.User{Bot: false},
			},
		},
	})

	if !called {
		t.Fatal("expected handler called with custom prefix")
	}
}

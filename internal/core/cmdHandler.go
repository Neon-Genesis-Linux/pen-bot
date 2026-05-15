package core

import (
	"strings"
	"sync"

	"github.com/disgoorg/disgo/events"
)

type MessageCommandHandler func(event *events.MessageCreate, args []string)

type commandNode struct {
	handler  MessageCommandHandler
	children map[string]*commandNode
}

var (
	rootCommand   = &commandNode{children: make(map[string]*commandNode)}
	registryMutex sync.RWMutex
	botPrefix     = "!" // configurable command prefix
)

// SetBotPrefix updates the command prefix at runtime.
func SetBotPrefix(prefix string) {
	botPrefix = prefix
}

// GetBotPrefix returns the active command prefix.
func GetBotPrefix() string {
	return botPrefix
}

// RegisterCommand registers a top-level command handler.
func RegisterCommand(command string, handler func(*events.MessageCreate)) {
	RegisterCommandPath([]string{command}, func(event *events.MessageCreate, args []string) {
		handler(event)
	})
}

// RegisterCommandPath registers a handler for a path of nested command names.
func RegisterCommandPath(path []string, handler MessageCommandHandler) {
	if len(path) == 0 || handler == nil {
		return
	}

	registryMutex.Lock()
	defer registryMutex.Unlock()

	node := rootCommand
	for _, segment := range path {
		if node.children == nil {
			node.children = make(map[string]*commandNode)
		}
		child, exists := node.children[segment]
		if !exists {
			child = &commandNode{children: make(map[string]*commandNode)}
			node.children[segment] = child
		}
		node = child
	}
	node.handler = handler
}

// RegisterSubcommand registers a nested handler under the given parent path.
func RegisterSubcommand(parent []string, command string, handler MessageCommandHandler) {
	path := append(append([]string(nil), parent...), command)
	RegisterCommandPath(path, handler)
}

// DispatchCommand dispatches a prefix-based message command to the deepest registered handler.
func DispatchCommand(event *events.MessageCreate) {
	if event.Message.Author.Bot {
		return
	}
	content := strings.TrimSpace(event.Message.Content)
	if !strings.HasPrefix(content, botPrefix) {
		return
	}
	words := strings.Fields(strings.TrimPrefix(content, botPrefix))
	if len(words) == 0 {
		return
	}

	registryMutex.RLock()
	defer registryMutex.RUnlock()

	node := rootCommand
	var lastHandler MessageCommandHandler
	var lastMatch int
	for i, word := range words {
		child, exists := node.children[word]
		if !exists {
			break
		}
		node = child
		if node.handler != nil {
			lastHandler = node.handler
			lastMatch = i + 1
		}
	}

	if lastHandler != nil {
		lastHandler(event, words[lastMatch:])
	}
}

// Slash commands in Disgo are handled separately via application command registration and interaction event listeners.

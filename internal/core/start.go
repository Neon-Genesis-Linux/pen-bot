package core

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/db"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
)

var (
	commandRegistry = make(map[string]func(*events.MessageCreate))
	registryMutex   sync.RWMutex
	botPrefix       = "!" // configurable command prefix
)

// SetBotPrefix sets the command prefix (default is "!")
func SetBotPrefix(prefix string) {
	botPrefix = prefix
}

// RegisterCommand registers a command handler
func RegisterCommand(command string, handler func(*events.MessageCreate)) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	commandRegistry[command] = handler
}

// DispatchCommand dispatches a message to the appropriate command handler
func DispatchCommand(event *events.MessageCreate) {
	if event.Message.Author.Bot {
		return
	}
	content := strings.TrimSpace(event.Message.Content)
	if !strings.HasPrefix(content, botPrefix) {
		return
	}
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}
	// Remove prefix from command to match registry key
	command := strings.TrimPrefix(parts[0], botPrefix)
	registryMutex.RLock()
	handler, exists := commandRegistry[command]
	registryMutex.RUnlock()
	if exists {
		handler(event)
	}
}

func Start(ctx context.Context, token string, listener func(*events.MessageCreate)) error {
	slog.Info("starting pen bot...")
	slog.Info("disgo version", slog.String("version", disgo.Version))

	client, err := disgo.New(token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent,
			),
		),
		bot.WithEventListenerFunc(listener),
	)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	if err = client.OpenGateway(ctx); err != nil {
		return err
	}

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	cleanupInterval := db.ParseCleanupIntervalEnv("DB_CACHE_CLEANUP_INTERVAL", 15)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("cache cleanup panicked", "recover", r)
			}
		}()
		db.StartCleanup(cleanupCtx, cleanupInterval)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("db connect panicked", "recover", r)
			}
		}()
		connectDBWithRetry(ctx)
	}()

	defer db.CloseDB()

	slog.Info("pen bot is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s:
		return nil
	}
}

func connectDBWithRetry(ctx context.Context) {
	var dbClient *db.DB
	var err error

	for attempt := range 10 {
		dbClient, err = db.NewFromEnv()
		if err == nil {
			break
		}
		slog.Warn("db connection attempt failed", "attempt", attempt+1, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
		}
	}
	if err != nil {
		slog.Error("db unavailable after retries", "error", err)
		return
	}

	if err := db.ApplyMigrations(ctx, dbClient); err != nil {
		slog.Error("db migration failed", "error", err)
		_ = dbClient.Close()
		return
	}

	db.SetGlobalDB(dbClient)
	slog.Info("db connected and ready")
}

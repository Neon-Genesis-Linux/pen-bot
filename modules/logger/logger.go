package logger

import (
	"log/slog"
	"os"
)

/*
	 init function that runs when the module is first imported.
		creates the custom logger with the declared handlers
*/
func init() {
	var handlers = buildHandlers()
	logger := slog.New(slog.NewMultiHandler(handlers...))
	slog.SetDefault(logger)
}

// builds slog handlers. Log destinations and config is set here
func buildHandlers() (handlers []slog.Handler) {

	// Default stdout logger. logs at INFO in production,
	// DEBUG otherwise.
	var logLevel slog.Level

	if (os.Getenv("ENV")) == "production" {
		logLevel = slog.LevelInfo
	} else {
		logLevel = slog.LevelDebug
	}
	// can also add a ReplaceAttr function for field modification here
	stdoutOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	}
	stdoutHandler := slog.NewTextHandler(os.Stdout, stdoutOpts)
	// Add new handlers when needed  in the same manner
	handlers = append(handlers, stdoutHandler)
	return
}

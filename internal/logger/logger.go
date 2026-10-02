// Package logger installs the process-wide slog handler on import, writing
// logfmt records to stdout at INFO when ENV is "production" and DEBUG otherwise.
// Each record carries the file and line it came from. Import it for its side
// effect.
package logger

import (
	"log/slog"
	"os"
)

func init() {
	var handlers = buildHandlers()
	logger := slog.New(slog.NewMultiHandler(handlers...))
	slog.SetDefault(logger)
}

func buildHandlers() (handlers []slog.Handler) {
	var logLevel slog.Level

	if (os.Getenv("ENV")) == "production" {
		logLevel = slog.LevelInfo
	} else {
		logLevel = slog.LevelDebug
	}

	stdoutOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	}
	stdoutHandler := slog.NewTextHandler(os.Stdout, stdoutOpts)
	handlers = append(handlers, stdoutHandler)
	return
}

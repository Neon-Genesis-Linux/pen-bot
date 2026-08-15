package logger

import (
	"go.uber.org/zap"
	"os"
	"sync"
)

var (
	instance *zap.Logger // singleton logger instance
	once     sync.Once   // sync primitive for ensuring logger init runs once
)

// creates a single logger instance for the module.
func ensureLogger() {
	once.Do(func() {
		var err error
		if os.Getenv("ENV") == "production" {
			instance, err = zap.NewProduction()
		} else {
			instance, err = zap.NewDevelopment()
		}
		if err != nil {
			panic(err)
		}
	})
}

/* These functions shadow Zap's loggers' calls,
while ensuring the singleton instance is initialized */

func Info(msg string, fields ...zap.Field) {
	ensureLogger()
	instance.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	ensureLogger()
	instance.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	ensureLogger()
	instance.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	ensureLogger()
	instance.Warn(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	ensureLogger()
	instance.Fatal(msg, fields...)
}

func With(fields ...zap.Field) *zap.Logger {
	ensureLogger()
	return instance.With(fields...)
}

func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

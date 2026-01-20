package logging

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	defaultLogger = NewLogger()
}

// NewLogger creates a new structured logger with JSON output
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Default returns the default logger instance
func Default() *slog.Logger {
	return defaultLogger
}

// WithComponent returns a logger with component name attached
func WithComponent(component string) *slog.Logger {
	return defaultLogger.With("component", component)
}

// Package logger provides a thin wrapper around the standard library
// structured logger (log/slog) so Core and modules share one logging style.
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON structured logger writing to stdout.
// level is one of: debug, info, warn, error (defaults to info).
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(handler)
}

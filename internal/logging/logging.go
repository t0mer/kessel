// Package logging builds the application slog.Logger from config.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// New returns a configured logger writing to stderr.
func New(level, format string) *slog.Logger {
	return newWithWriter(os.Stderr, level, format)
}

func newWithWriter(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: levelFromString(level)}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}

func levelFromString(s string) slog.Leveler {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

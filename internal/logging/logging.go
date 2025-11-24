package logging

import (
	"log/slog"
	"os"
)

// New creates a slog.Logger configured for CLI usage.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

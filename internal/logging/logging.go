package logging

import (
	"log/slog"
	"os"
)

// Setup installs the default slog logger. Writes RFC3339 timestamps and
// key=value attributes to stderr. Level is INFO unless verbose is true,
// in which case it's DEBUG.
func Setup(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

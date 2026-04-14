package logger

import (
	"log/slog"
	"os"
)

// InitLogger initializes the global slog instance based on the environment.
// It sets up a JSON logger for production or a Text logger for development.
func InitLogger(isProd bool) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}

	if isProd {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

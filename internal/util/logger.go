package util

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

// SetupLogger initializes the global structured logger
func SetupLogger(environment string) {
	var handler slog.Handler

	if environment == "production" {
		// JSON format for production environments
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		// Text format for development/local environments
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

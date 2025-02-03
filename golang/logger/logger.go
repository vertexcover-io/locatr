package logger

import (
	"log/slog"
	"os"
)

var Level = new(slog.LevelVar)

func newLogger() *slog.Logger {
	Level.Set(slog.LevelError)
	opts := &slog.HandlerOptions{
		Level: Level,
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

var Logger = newLogger()

package logger

import (
	"log/slog"
	"os"
)

func newLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelError,
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

var Logger = newLogger()

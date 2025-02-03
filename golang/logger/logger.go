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
	log_file, err := os.Create("locator_log.json")
	if err != nil {
		panic("failed to create log file")
	}
	return slog.New(slog.NewJSONHandler(log_file, opts))
}

var Logger = newLogger()

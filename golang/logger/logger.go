package logger

import (
	"log/slog"
	"os"
	"time"
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

func GetTimeLogger(topic string) func() {
	start_time := time.Now()
	return func() {
		time_taken := time.Since(start_time)
		Logger.Debug("time taken for", slog.String("topic", topic), slog.Int64("milliseconds", time_taken.Milliseconds()))
	}
}

var Logger = newLogger()

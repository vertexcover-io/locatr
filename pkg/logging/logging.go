package logging

import (
	"io"
	"log/slog"
	"os"
	"time"
)

var slogLevel = new(slog.LevelVar)

// NewLogger creates a new logger instance with a default log level of Debug.
func NewLogger(level slog.Level, writer io.Writer) *slog.Logger {
	slogLevel.Set(level)
	return slog.New(
		slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slogLevel}),
	)
}

// CreateTopic starts a timer for a given topic and logs the time taken for the topic.
func CreateTopic(topic string, logger *slog.Logger) func() {
	logger.Debug("Starting", slog.String("topic", topic))
	startTime := time.Now()
	return func() {
		logger.Debug(
			"Time Elapsed",
			slog.String("topic", topic),
			slog.Float64("seconds", time.Since(startTime).Seconds()),
		)
	}
}

// Logger instance with Debug level that writes logs to standard output.
var DefaultLogger = NewLogger(slog.LevelDebug, os.Stdout)

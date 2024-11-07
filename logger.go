package locatr

import (
	"log"
	"os"
)

const (
	// Silent will not log anything
	Silent LogLevel = iota + 1
	// Error will log only errors
	Error
	// Warn will log errors and warnings
	Warn
	// Info will log errors, warnings and info
	Info
	// Debug will log everything
	Debug
)

var (
	infoStr  = "INFO: %s"
	warnStr  = "WARN: %s"
	errorStr = "ERROR: %s"
	debugStr = "DEBUG: %s"
)

type Writer interface {
	Printf(string, ...interface{})
}

type LogConfig struct {
	// Level is the log level
	Level LogLevel
	// Writer is the writer to write logs to
	Writer Writer
}

type logInterface interface {
	LogMode(LogLevel) logInterface
	Info(string)
	Warn(string)
	Error(string)
	Debug(string)
}

type logger struct {
	config LogConfig
	Writer
}

func (l *logger) LogMode(level LogLevel) logInterface {
	l.config.Level = level
	return l
}

func (l *logger) log(level LogLevel, formatStr, message string) {
	if l.config.Level >= level {
		l.Printf(formatStr, message)
	}
}

func (l *logger) Info(message string) {
	l.log(Info, infoStr, message)
}

func (l *logger) Warn(message string) {
	l.log(Warn, warnStr, message)
}

func (l *logger) Error(message string) {
	l.log(Error, errorStr, message)
}

func (l *logger) Debug(message string) {
	l.log(Debug, debugStr, message)
}

var DefaultLogWriter = log.New(os.Stdout, "\n", log.LstdFlags)

func NewLogger(config LogConfig) logInterface {
	if config.Level == 0 {
		config.Level = Error
	}
	return &logger{
		config: config,
		Writer: config.Writer,
	}
}

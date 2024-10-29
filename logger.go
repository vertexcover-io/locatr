package locatr

import (
	"log"
	"os"
)

type LogLevel int

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

type LogConfig struct {
	// Level is the log level
	Level LogLevel
}

type logInterface interface {
	LogMode(LogLevel) logInterface
	Info(string)
	Warn(string)
	Error(string)
	Debug(string)
}
type Writer interface {
	Printf(string, ...interface{})
}

type logger struct {
	config                               LogConfig
	infoStr, warnStr, errorStr, debugStr string
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
	l.log(Info, l.infoStr, message)
}

func (l *logger) Warn(message string) {
	l.log(Warn, l.warnStr, message)
}

func (l *logger) Error(message string) {
	l.log(Error, l.errorStr, message)
}

func (l *logger) Debug(message string) {
	l.log(Debug, l.debugStr, message)
}

func NewLogger(config LogConfig) logInterface {
	var (
		infoStr  = "INFO: %s"
		warnStr  = "WARN: %s"
		errorStr = "ERROR: %s"
		debugStr = "DEBUG: %s"
	)
	if config.Level == 0 {
		config.Level = Info
	}
	return &logger{
		config:   config,
		infoStr:  infoStr,
		warnStr:  warnStr,
		errorStr: errorStr,
		debugStr: debugStr,
		Writer:   log.New(os.Stdout, "\n", log.LstdFlags),
	}
}

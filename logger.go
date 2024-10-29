package locatr

import (
	"fmt"
	"time"
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

type logger struct {
	config                               LogConfig
	infoStr, warnStr, errorStr, debugStr string
}

func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (l *logger) LogMode(level LogLevel) logInterface {
	l.config.Level = level
	return l
}

func (l *logger) log(level LogLevel, formatStr, message string) {
	if l.config.Level >= level {
		logMessage := fmt.Sprintf(formatStr, getCurrentTime(), message)
		fmt.Println(logMessage)
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
		infoStr  = "INFO: %s %s"
		warnStr  = "WARN: %s %s"
		errorStr = "ERROR: %s %s"
		debugStr = "DEBUG: %s %s"
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
	}
}

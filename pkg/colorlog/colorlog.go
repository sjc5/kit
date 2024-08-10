package colorlog

import (
	"fmt"
	"log"
)

const (
	colorReset = "\033[0m"
	colorInfo  = "\033[36m" // Light blue
	colorWarn  = "\033[33m" // Yellow
	colorError = "\033[31m" // Red
)

type Logger interface {
	Info(args ...any)
	Infof(format string, args ...any)
	Warning(args ...any)
	Warningf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
}

type Log struct {
	Label string
}

func (l *Log) log(level string, args ...any) {
	log.Printf(" %s %s %v%s\n", l.Label, levelToColor(level), args, colorReset)
}

func (l *Log) logf(level, format string, args ...any) {
	log.Printf(" %s %s %s%s\n", l.Label, levelToColor(level), fmt.Sprintf(format, args...), colorReset)
}

func levelToColor(level string) string {
	switch level {
	case "info":
		return colorInfo
	case "warning":
		return colorWarn
	case "error":
		return colorError
	default:
		return ""
	}
}

func (l *Log) Info(args ...any) {
	l.log("info", args...)
}

func (l *Log) Infof(format string, args ...any) {
	l.logf("info", format, args...)
}

func (l *Log) Warning(args ...any) {
	l.log("warning", args...)
}

func (l *Log) Warningf(format string, args ...any) {
	l.logf("warning", format, args...)
}

func (l *Log) Error(args ...any) {
	l.log("error", args...)
}

func (l *Log) Errorf(format string, args ...any) {
	l.logf("error", format, args...)
}

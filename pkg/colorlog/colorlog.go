package colorlog

import (
	"fmt"
	"log"
)

type Log struct {
	Label string
}

func (l *Log) log(level string, args ...any) {
	colorCode := levelToColor(level)
	labelToUse := mungeLabel(l.Label)
	log.Printf(" %s %s %v\n", labelToUse, colorCode, args)
	resetColor()
}

func (l *Log) logf(level, format string, args ...any) {
	colorCode := levelToColor(level)
	labelToUse := mungeLabel(l.Label)
	log.Printf(" %s %s %s\n", labelToUse, colorCode, fmt.Sprintf(format, args...))
	resetColor()
}

func levelToColor(level string) string {
	switch level {
	case "info":
		return "\033[36m" // Light blue
	case "warning":
		return "\033[33m" // Yellow
	case "error":
		return "\033[31m" // Red
	default:
		return ""
	}
}

func mungeLabel(label string) string {
	if len(label) < 6 {
		return fmt.Sprintf("%-6s", label)
	}
	return label
}

func resetColor() {
	fmt.Print("\033[0m")
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

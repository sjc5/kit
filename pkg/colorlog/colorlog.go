package colorlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const (
	colorReset = "\033[0m"
	colorDebug = "\033[37m" // Light gray
	colorInfo  = "\033[36m" // Light blue
	colorWarn  = "\033[33m" // Yellow
	colorError = "\033[31m" // Red
)

type ColorLogHandler struct {
	label  string
	output io.Writer
}

func New(label string) *slog.Logger {
	handler := &ColorLogHandler{label: label, output: os.Stdout}
	return slog.New(handler)
}

func (h *ColorLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *ColorLogHandler) Handle(_ context.Context, r slog.Record) error {
	color := h.levelToColor(r.Level)

	// Format time in a similar way to log.Printf
	timeStr := r.Time.Format("2006/01/02 15:04:05")

	// Convert attributes to a map for easier handling
	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	hasAttrs := len(attrs) > 0

	attrsStr := ""
	if hasAttrs {
		count := 0
		for k, v := range attrs {
			count++
			attrsStr += fmt.Sprintf("%s[%s %s%s=%s%v %s]%s", colorDebug, colorReset, k, colorDebug, colorReset, v, colorDebug, colorReset)
			if count < len(attrs) {
				attrsStr += " "
			}
		}
	}

	// Format the message with attributes
	var msg string
	if !hasAttrs {
		msg = fmt.Sprintf("%s  %s  %s%s%s\n",
			timeStr,
			h.label,
			color,
			r.Message,
			colorReset,
		)
	} else {
		msg = fmt.Sprintf("%s  %s  %s%s %s%s\n",
			timeStr,
			h.label,
			color,
			r.Message,
			colorReset,
			attrsStr,
		)
	}

	_, err := fmt.Fprint(h.output, msg)
	return err
}

func (h *ColorLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ColorLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *ColorLogHandler) levelToColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return colorError
	case level >= slog.LevelWarn:
		return colorWarn
	case level >= slog.LevelInfo:
		return colorInfo
	case level >= slog.LevelDebug:
		return colorDebug
	default:
		return colorDebug
	}
}

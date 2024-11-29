package colorlog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestColorLogHandler_Levels(t *testing.T) {
	var buf bytes.Buffer
	logger := New("[TEST]")
	logger.Handler().(*ColorLogHandler).output = &buf

	tests := []struct {
		name     string
		logFn    func(string, ...any)
		message  string
		color    string
		wantAttr bool
	}{
		{"Debug", logger.Debug, "debug message", colorDebug, false},
		{"Info", logger.Info, "info message", colorInfo, false},
		{"Warn", logger.Warn, "warning message", colorWarn, false},
		{"Error", logger.Error, "error message", colorError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn(tt.message)
			got := buf.String()

			// Check color codes
			if !strings.Contains(got, tt.color) {
				t.Errorf("expected output to contain color %q, got %q", tt.color, got)
			}

			// Check message content
			if !strings.Contains(got, tt.message) {
				t.Errorf("expected output to contain message %q, got %q", tt.message, got)
			}

			// Check label
			if !strings.Contains(got, "[TEST]") {
				t.Errorf("expected output to contain label [TEST], got %q", got)
			}

			// Check color reset
			if !strings.Contains(got, colorReset) {
				t.Errorf("expected output to contain reset color code, got %q", got)
			}
		})
	}
}

func TestColorLogHandler_WithAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := New("[TEST]")
	logger.Handler().(*ColorLogHandler).output = &buf

	logger.Info("test message", "key1", "value1", "key2", 42)
	got := buf.String()

	// Check for key elements in the attribute formatting
	expectations := []string{
		colorDebug + "[" + colorReset + " key1" + colorDebug + "=" + colorReset + "value1 " + colorDebug + "]" + colorReset,
		colorDebug + "[" + colorReset + " key2" + colorDebug + "=" + colorReset + "42 " + colorDebug + "]" + colorReset,
	}

	for _, exp := range expectations {
		if !strings.Contains(got, exp) {
			t.Errorf("expected output to contain %q, got %q", exp, got)
		}
	}
}

func TestColorLogHandler_TimeFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New("[TEST]")
	logger.Handler().(*ColorLogHandler).output = &buf

	logger.Info("test message")
	got := buf.String()

	// Check time format (2006/01/02 15:04:05)
	timeStr := time.Now().Format("2006/01/02")
	if !strings.Contains(got, timeStr) {
		t.Errorf("expected output to contain time format %q, got %q", timeStr, got)
	}
}

func TestColorLogHandler_Interface(t *testing.T) {
	handler := &ColorLogHandler{}

	// Test Enabled method
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled should return true")
	}

	// Test WithAttrs method
	newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})
	if newHandler != handler {
		t.Error("WithAttrs should return the same handler")
	}

	// Test WithGroup method
	newHandler = handler.WithGroup("group")
	if newHandler != handler {
		t.Error("WithGroup should return the same handler")
	}
}

func TestNew(t *testing.T) {
	logger := New("[TEST]")
	if logger == nil {
		t.Error("New should not return nil")
	}

	handler, ok := logger.Handler().(*ColorLogHandler)
	if !ok {
		t.Error("logger.Handler should be of type *ColorLogHandler")
	}

	if handler.label != "[TEST]" {
		t.Errorf("handler label should be [TEST], got %q", handler.label)
	}
}

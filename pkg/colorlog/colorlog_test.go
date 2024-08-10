package colorlog

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLog_Info(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0) // Disable log prefix for testing

	logger := &Log{Label: "TEST"}
	logger.Info("This is an info message")

	expected := " TEST \033[36m [This is an info message]\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_Infof(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Infof("This is an %s message", "info")

	expected := " TEST \033[36m This is an info message\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_Warning(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Warning("This is a warning message")

	expected := " TEST \033[33m [This is a warning message]\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_Warningf(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Warningf("This is a %s message", "warning")

	expected := " TEST \033[33m This is a warning message\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_Error(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Error("This is an error message")

	expected := " TEST \033[31m [This is an error message]\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_Errorf(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Errorf("This is an %s message", "error")

	expected := " TEST \033[31m This is an error message\033[0m\n"
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("expected log output to contain %q, got %q", expected, buf.String())
	}
}

func TestLog_ResetColor(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := &Log{Label: "TEST"}
	logger.Info("This is an info message")
	logger.Warning("This is a warning message")
	logger.Error("This is an error message")

	output := buf.String()
	if !strings.HasSuffix(output, "\033[0m\n") {
		t.Fatalf("expected log output to end with reset color code, got %q", output)
	}
}

package timer

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func captureOutput(f func()) string {
	// Create a pipe to capture standard output
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	// Run the function that writes to standard output
	f()

	// Close the writer and reset os.Stdout
	w.Close()
	os.Stdout = stdout

	// Read the captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNewTimer(t *testing.T) {
	tm := New()
	if !tm.on {
		t.Error("expected timer to be on")
	}
	if tm.start.IsZero() {
		t.Error("expected timer start time to be set")
	}
}

func TestConditionalTimer(t *testing.T) {
	t.Run("ConditionTrue", func(t *testing.T) {
		tm := Conditional(true)
		if !tm.on {
			t.Error("expected timer to be on when condition is true")
		}
		if tm.start.IsZero() {
			t.Error("expected timer start time to be set when condition is true")
		}
	})

	t.Run("ConditionFalse", func(t *testing.T) {
		tm := Conditional(false)
		if tm.on {
			t.Error("expected timer to be off when condition is false")
		}
		if tm.start.IsZero() {
			t.Error("expected timer start time to be set even if condition is false")
		}
	})
}

func TestCheckpoint(t *testing.T) {
	tm := New()

	time.Sleep(10 * time.Millisecond) // simulate work
	output := captureOutput(func() {
		tm.Checkpoint("test")
	})

	if output == "" {
		t.Error("expected output from active timer checkpoint")
	}
	if output[:8] != "duration" {
		t.Error("expected checkpoint label in output")
	}
}

func TestReset(t *testing.T) {
	tm := New()
	time.Sleep(10 * time.Millisecond)
	oldStart := tm.start
	tm.Reset()

	if tm.start == oldStart {
		t.Error("expected timer start time to be reset")
	}

	if time.Since(tm.start) >= 10*time.Millisecond {
		t.Error("expected timer start time to be close to current time after reset")
	}
}

func TestMultipleCheckpoints(t *testing.T) {
	tm := New()

	time.Sleep(5 * time.Millisecond)
	output1 := captureOutput(func() {
		tm.Checkpoint("checkpoint1")
	})

	time.Sleep(5 * time.Millisecond)
	output2 := captureOutput(func() {
		tm.Checkpoint("checkpoint2")
	})

	if output1 == output2 {
		t.Error("expected different durations for consecutive checkpoints")
	}
}

func TestConditionalTimerDoesNotResetWhenOff(t *testing.T) {
	tm := Conditional(false)
	oldStart := tm.start

	time.Sleep(10 * time.Millisecond)
	tm.Reset()

	if tm.start != oldStart {
		t.Error("expected timer start time to remain unchanged when off")
	}
}

func TestResetImmediate(t *testing.T) {
	tm := New()
	tm.Reset() // Immediate reset after creation

	if time.Since(tm.start) >= 10*time.Millisecond {
		t.Error("expected timer start time to be close to current time after immediate reset")
	}
}

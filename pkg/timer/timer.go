package timer

import (
	"fmt"
	"time"
)

type timer struct {
	start time.Time
	on    bool
}

// New returns a new timer.
func New() *timer {
	return &timer{start: time.Now(), on: true}
}

// Conditional returns a new timer that is only active if the condition is true.
func Conditional(condition bool) *timer {
	return &timer{start: time.Now(), on: condition}
}

// Checkpoint prints the duration since the last checkpoint and resets the timer.
func (t *timer) Checkpoint(label string) {
	if !t.on {
		return
	}
	fmt.Println("duration for", label, ":", time.Since(t.start))
	t.Reset()
}

// Reset resets the timer.
func (t *timer) Reset() {
	t.start = time.Now()
}

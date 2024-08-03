package id

import (
	"strings"
	"testing"
)

func TestIDNew(t *testing.T) {
	for i := range 255 {
		id, err := New(uint8(i))

		// ensure no error
		if err != nil {
			t.Errorf("New() returned error: %v", err)
			return
		}

		// ensure correct length
		if len(id) != i {
			t.Errorf("New() returned ID of length %d, expected %d", len(id), i)
			return
		}

		// ensure no invalid characters
		if strings.ContainsAny(id, " -_+/=") {
			t.Errorf("New() returned ID with invalid characters: %s", id)
			return
		}
	}
}

package bytesutil

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// TestStruct is a simple structure used for testing gob encoding and decoding.
type TestStruct struct {
	Name string
	Age  int
}

func TestRandom(t *testing.T) {
	// Test generating random bytes
	byteLen := 16
	randomBytes, err := Random(byteLen)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(randomBytes) != byteLen {
		t.Fatalf("expected random byte slice of length %d, got %d", byteLen, len(randomBytes))
	}

	// Test randomness by generating another set and comparing
	anotherRandomBytes, err := Random(byteLen)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if bytes.Equal(randomBytes, anotherRandomBytes) {
		t.Fatalf("expected different random byte slices, got identical slices")
	}
}

func TestFromBase64(t *testing.T) {
	// Test decoding a valid base64 string
	originalBytes := []byte("test message")
	base64Str := base64.StdEncoding.EncodeToString(originalBytes)

	decodedBytes, err := FromBase64(base64Str)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(decodedBytes, originalBytes) {
		t.Fatalf("expected decoded bytes to match original bytes")
	}

	// Test decoding an invalid base64 string
	_, err = FromBase64("invalid base64")
	if err == nil {
		t.Fatalf("expected error for invalid base64 string, got nil")
	}
}

func TestToBase64(t *testing.T) {
	// Test encoding bytes to base64
	originalBytes := []byte("test message")
	base64Str := ToBase64(originalBytes)

	expectedBase64Str := base64.StdEncoding.EncodeToString(originalBytes)
	if base64Str != expectedBase64Str {
		t.Fatalf("expected base64-encoded string to be %s, got %s", expectedBase64Str, base64Str)
	}
}

func TestToGob(t *testing.T) {
	// Test encoding a struct to gob
	originalStruct := TestStruct{Name: "John", Age: 30}

	gobBytes, err := ToGob(originalStruct)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(gobBytes) == 0 {
		t.Fatalf("expected gob-encoded byte slice to be non-empty")
	}
}

func TestFromGobInto(t *testing.T) {
	// Test decoding a gob-encoded byte slice into a struct
	originalStruct := TestStruct{Name: "John", Age: 30}

	gobBytes, err := ToGob(originalStruct)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decodedStruct TestStruct
	err = FromGobInto(gobBytes, &decodedStruct)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if originalStruct != decodedStruct {
		t.Fatalf("expected decoded struct to match original struct")
	}

	// Test decoding a nil byte slice
	err = FromGobInto(nil, &decodedStruct)
	if err == nil {
		t.Fatalf("expected error for nil gobBytes, got nil")
	}

	// Test decoding into a nil destination
	err = FromGobInto(gobBytes, nil)
	if err == nil {
		t.Fatalf("expected error for nil destination, got nil")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test Random with a byte length of 0
	zeroLengthBytes, err := Random(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(zeroLengthBytes) != 0 {
		t.Fatalf("expected empty byte slice, got %d bytes", len(zeroLengthBytes))
	}

	// Test FromBase64 with an empty string
	emptyBytes, err := FromBase64("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(emptyBytes) != 0 {
		t.Fatalf("expected empty byte slice, got %d bytes", len(emptyBytes))
	}

	// Test ToBase64 with an empty byte slice
	emptyBase64Str := ToBase64([]byte{})
	if emptyBase64Str != "" {
		t.Fatalf("expected empty base64 string, got %s", emptyBase64Str)
	}

	// Test ToGob with a nil source
	_, err = ToGob(nil)
	if err == nil {
		t.Fatalf("expected error for nil source, got nil")
	}

	// Test FromGobInto with empty gob bytes
	var decodedStruct TestStruct
	err = FromGobInto([]byte{}, &decodedStruct)
	if err == nil {
		t.Fatalf("expected error for empty gob bytes, got nil")
	}
}

package bytesutil

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
)

// Random returns a slice of cryptographically random bytes of length byteLen.
func Random(byteLen int) ([]byte, error) {
	r := make([]byte, byteLen)
	if _, err := rand.Read(r); err != nil {
		return nil, err
	}
	return r, nil
}

// FromBase64 decodes a base64-encoded string into a byte slice.
func FromBase64(base64Str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64Str)
}

// ToBase64 encodes a byte slice into a base64-encoded string.
func ToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// ToGob encodes an arbitrary value into a gob-encoded byte slice.
func ToGob(src any) ([]byte, error) {
	var a bytes.Buffer
	enc := gob.NewEncoder(&a)
	err := enc.Encode(src)
	if err != nil {
		return nil, fmt.Errorf("bytesutil.ToGob: failed to encode src to bytes: %w", err)
	}
	return a.Bytes(), nil
}

// FromGob decodes a gob-encoded byte slice into an arbitrary value.
func FromGobInto(gobBytes []byte, dest any) error {
	if gobBytes == nil {
		return fmt.Errorf("bytesutil.FromGobInto: cannot decode nil bytes")
	}
	dec := gob.NewDecoder(bytes.NewReader(gobBytes))
	err := dec.Decode(dest)
	if err != nil {
		return fmt.Errorf("failed to decode bytes into dest: %w", err)
	}
	return nil
}

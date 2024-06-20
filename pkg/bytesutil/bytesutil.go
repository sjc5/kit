package bytesutil

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
)

func Random(l int) ([]byte, error) {
	r := make([]byte, l)
	if _, err := rand.Read(r); err != nil {
		return nil, err
	}
	return r, nil
}

func FromBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func ToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func ToGob(src any) ([]byte, error) {
	var a bytes.Buffer
	enc := gob.NewEncoder(&a)
	err := enc.Encode(src)
	if err != nil {
		return nil, fmt.Errorf("bytesutil.ToGob: failed to encode src to bytes: %w", err)
	}
	return a.Bytes(), nil
}

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

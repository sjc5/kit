package bytesutil

import (
	"crypto/rand"
	"encoding/base64"
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

package cryptoutil

import (
	"crypto/rand"
	"fmt"
)

func RandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("error generating random key: %v", err)
	}
	return key, nil
}

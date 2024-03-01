package jsonutil

import (
	"encoding/json"
	"fmt"
)

func ToString(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error encoding JSON: %w", err)
	}
	return string(b), nil
}

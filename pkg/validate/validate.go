package validate

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type Validate struct {
	Instance *validator.Validate
}

func (v Validate) UnmarshalFromRequest(r *http.Request, destination interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(destination); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(destination); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

func (v Validate) UnmarshalFromString(data string, destination interface{}) error {
	if err := json.Unmarshal([]byte(data), destination); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(destination); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

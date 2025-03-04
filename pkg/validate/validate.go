// Package validate provides a simple way to validate and parse data from HTTP requests.
package validate

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Validate is a wrapper around the go-playground/validator package.
// It provides methods for validating and parsing data from HTTP requests.
type Validate struct {
	Instance *validator.Validate
}

// New creates a new Validate instance.
func New() *Validate {
	return &Validate{
		Instance: validator.New(validator.WithRequiredStructEnabled()),
	}
}

const ValidationErrorPrefix = "validation error: "

// IsValidationError returns true if the error is a validation error.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	if len(errMsg) < len(ValidationErrorPrefix) {
		return false
	}
	return errMsg[:len(ValidationErrorPrefix)] == ValidationErrorPrefix
}

// JSONBodyInto decodes an HTTP request body into a struct and validates it.
func (v Validate) JSONBodyInto(r *http.Request, destStructPtr any) error {
	if err := json.NewDecoder(r.Body).Decode(destStructPtr); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(destStructPtr); err != nil {
		return fmt.Errorf(ValidationErrorPrefix+"%w", err)
	}
	return nil
}

// JSONBytesInto decodes a byte slice containing JSON data into a struct and validates it.
func (v Validate) JSONBytesInto(data []byte, destStructPtr any) error {
	if err := json.Unmarshal(data, destStructPtr); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(destStructPtr); err != nil {
		return fmt.Errorf(ValidationErrorPrefix+"%w", err)
	}
	return nil
}

// JSONStrInto decodes a string containing JSON data into a struct and validates it.
func (v Validate) JSONStrInto(data string, destStructPtr any) error {
	return v.JSONBytesInto([]byte(data), destStructPtr)
}

// URLSearchParamsInto parses the URL parameters of an HTTP request into a struct and validates it.
func (v Validate) URLSearchParamsInto(r *http.Request, destStructPtr any) error {
	err := parseURLValues(r.URL.Query(), destStructPtr)
	if err != nil {
		return fmt.Errorf("error parsing URL parameters: %w", err)
	}
	if err := v.Instance.Struct(destStructPtr); err != nil {
		return fmt.Errorf(ValidationErrorPrefix+"%w", err)
	}
	return nil
}

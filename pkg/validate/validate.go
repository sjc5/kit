// Package validate provides a simple way to validate and parse data from HTTP requests.
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Validate is a wrapper around the go-playground/validator package.
// It provides methods for validating and parsing data from HTTP requests.
type Validate struct {
	Instance *validator.Validate
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

// JSONBodyInto decodes the JSON body of an HTTP request into a struct and validates it.
func (v Validate) JSONBodyInto(body io.ReadCloser, destStructPtr any) error {
	if err := json.NewDecoder(body).Decode(destStructPtr); err != nil {
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

// Deprecated: UnmarshalFromRequest is deprecated. Use `v.JSONBodyInto(r.Body, dest)` instead.
func (v Validate) UnmarshalFromRequest(r *http.Request, destStructPtr any) error {
	return v.JSONBodyInto(r.Body, destStructPtr)
}

// Deprecated: UnmarshalFromBytes is deprecated. Use JSONBytesInto instead.
func (v Validate) UnmarshalFromBytes(data []byte, destStructPtr any) error {
	return v.JSONBytesInto(data, destStructPtr)
}

// Deprecated: UnmarshalFromString is deprecated. Use JSONStrInto instead.
func (v Validate) UnmarshalFromString(data string, destStructPtr any) error {
	return v.JSONStrInto(data, destStructPtr)
}

// Deprecated: UnmarshalFromResponse is deprecated. Use `v.JSONBodyInto(r.Body, dest)` instead.
func (v Validate) UnmarshalFromResponse(r *http.Response, destStructPtr any) error {
	return v.JSONBodyInto(r.Body, destStructPtr)
}

package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type Validate struct {
	Instance *validator.Validate
}

func (v Validate) JSONBodyInto(body io.ReadCloser, dest any) error {
	if err := json.NewDecoder(body).Decode(dest); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(dest); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

func (v Validate) JSONBytesInto(data []byte, dest any) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}
	if err := v.Instance.Struct(dest); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

func (v Validate) JSONStrInto(data string, dest any) error {
	return v.UnmarshalFromBytes([]byte(data), dest)
}

func (v Validate) URLSearchParamsInto(r *http.Request, dest any) error {
	err := parseURLValues(r.URL.Query(), dest)
	if err != nil {
		return fmt.Errorf("error parsing URL parameters: %w", err)
	}
	if err := v.Instance.Struct(dest); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

// Deprecated: UnmarshalFromRequest is deprecated. Use `v.JSONBodyInto(r.Body, dest)` instead.
func (v Validate) UnmarshalFromRequest(r *http.Request, dest any) error {
	return v.JSONBodyInto(r.Body, dest)
}

// Deprecated: UnmarshalFromBytes is deprecated. Use JSONBytesInto instead.
func (v Validate) UnmarshalFromBytes(data []byte, dest any) error {
	return v.JSONBytesInto(data, dest)
}

// Deprecated: UnmarshalFromString is deprecated. Use JSONStrInto instead.
func (v Validate) UnmarshalFromString(data string, dest any) error {
	return v.JSONStrInto(data, dest)
}

// Deprecated: UnmarshalFromResponse is deprecated. Use `v.JSONBodyInto(r.Body, dest)` instead.
func (v Validate) UnmarshalFromResponse(r *http.Response, dest any) error {
	return v.JSONBodyInto(r.Body, dest)
}

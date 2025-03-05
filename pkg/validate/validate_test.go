package validate

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type TestStruct struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=18"`
}

func TestIsValidationError(t *testing.T) {
	// Test with nil error
	if IsValidationError(nil) {
		t.Error("expected false, got true for nil error")
	}

	// Test with a non-validation error
	nonValidationError := errors.New("some other error")
	if IsValidationError(nonValidationError) {
		t.Error("expected false, got true for non-validation error")
	}

	// Test with a validation error
	validationError := errors.New(ValidationErrorPrefix + "field is required")
	if !IsValidationError(validationError) {
		t.Error("expected true, got false for validation error")
	}
}

func TestJSONBodyInto(t *testing.T) {
	v := New()

	// Test with valid JSON
	validJSON := `{"name": "John", "email": "john@example.com", "age": 30}`
	req := &http.Request{Body: io.NopCloser(strings.NewReader(validJSON))}
	dest := &TestStruct{}
	if err := v.JSONBodyInto(req, dest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dest.Name != "John" || dest.Email != "john@example.com" || dest.Age != 30 {
		t.Error("unexpected values in struct after decoding")
	}

	// Test with invalid JSON
	invalidJSON := `{"name": "John", "email": "john@example.com"`
	req = &http.Request{Body: io.NopCloser(strings.NewReader(invalidJSON))}
	dest = &TestStruct{}
	err := v.JSONBodyInto(req, dest)
	if err == nil || !strings.Contains(err.Error(), "error decoding JSON") {
		t.Errorf("expected decoding error, got %v", err)
	}

	// Test with missing required fields
	missingFieldsJSON := `{"name": "John"}`
	req = &http.Request{Body: io.NopCloser(strings.NewReader(missingFieldsJSON))}
	dest = &TestStruct{}
	err = v.JSONBodyInto(req, dest)
	if err == nil || !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestJSONBytesInto(t *testing.T) {
	v := New()

	// Test with valid JSON
	validJSON := []byte(`{"name": "John", "email": "john@example.com", "age": 30}`)
	dest := &TestStruct{}
	if err := v.JSONBytesInto(validJSON, dest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dest.Name != "John" || dest.Email != "john@example.com" || dest.Age != 30 {
		t.Error("unexpected values in struct after decoding")
	}

	// Test with invalid JSON
	invalidJSON := []byte(`{"name": "John", "email": "john@example.com"`)
	dest = &TestStruct{}
	err := v.JSONBytesInto(invalidJSON, dest)
	if err == nil || !strings.Contains(err.Error(), "error decoding JSON") {
		t.Errorf("expected decoding error, got %v", err)
	}

	// Test with missing required fields
	missingFieldsJSON := []byte(`{"name": "John"}`)
	dest = &TestStruct{}
	err = v.JSONBytesInto(missingFieldsJSON, dest)
	if err == nil || !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestJSONStrInto(t *testing.T) {
	v := New()

	// Test with valid JSON
	validJSON := `{"name": "John", "email": "john@example.com", "age": 30}`
	dest := &TestStruct{}
	if err := v.JSONStrInto(validJSON, dest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dest.Name != "John" || dest.Email != "john@example.com" || dest.Age != 30 {
		t.Error("unexpected values in struct after decoding")
	}

	// Test with invalid JSON
	invalidJSON := `{"name": "John", "email": "john@example.com"`
	dest = &TestStruct{}
	err := v.JSONStrInto(invalidJSON, dest)
	if err == nil || !strings.Contains(err.Error(), "error decoding JSON") {
		t.Errorf("expected decoding error, got %v", err)
	}

	// Test with missing required fields
	missingFieldsJSON := `{"name": "John"}`
	dest = &TestStruct{}
	err = v.JSONStrInto(missingFieldsJSON, dest)
	if err == nil || !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestURLSearchParamsIntoHighLevel(t *testing.T) {
	v := New()

	// Test with valid URL parameters
	urlParams := url.Values{}
	urlParams.Add("name", "John")
	urlParams.Add("email", "john@example.com")
	urlParams.Add("age", "30")
	req := &http.Request{URL: &url.URL{RawQuery: urlParams.Encode()}}
	dest := &TestStruct{}
	if err := v.URLSearchParamsInto(req, dest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dest.Name != "John" || dest.Email != "john@example.com" || dest.Age != 30 {
		t.Error("unexpected values in struct after parsing URL parameters")
	}

	// Test with missing required fields
	urlParams = url.Values{}
	urlParams.Add("name", "John")
	req = &http.Request{URL: &url.URL{RawQuery: urlParams.Encode()}}
	dest = &TestStruct{}
	err := v.URLSearchParamsInto(req, dest)
	if err == nil || !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestEdgeCases(t *testing.T) {
	v := New()

	// Test with empty JSON
	emptyJSON := `{}`
	dest := &TestStruct{}
	err := v.JSONStrInto(emptyJSON, dest)
	if err == nil || !IsValidationError(err) {
		t.Errorf("expected validation error, got %v", err)
	}

	// Test with unexpected field type
	wrongTypeJSON := `{"name": "John", "email": "john@example.com", "age": "not a number"}`
	err = v.JSONStrInto(wrongTypeJSON, dest)
	if err == nil || !strings.Contains(err.Error(), "error decoding JSON") {
		t.Errorf("expected decoding error, got %v", err)
	}

	// Test with large payload
	largePayload := strings.Repeat(`{"name": "John", "email": "john@example.com", "age": 30}`, 10000)
	dest = &TestStruct{}
	err = v.JSONStrInto(largePayload, dest)
	if err == nil {
		t.Errorf("expected error due to large payload, got nil")
	}

	// Test with special characters in JSON
	specialCharJSON := `{"name": "J@hn!$#", "email": "john@example.com", "age": 30}`
	err = v.JSONStrInto(specialCharJSON, dest)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dest.Name != "J@hn!$#" {
		t.Errorf("unexpected name value, got %s", dest.Name)
	}
}

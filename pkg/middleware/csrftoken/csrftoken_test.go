package csrftoken

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock functions
func mockGetExpectedCSRFToken(expectedToken Token, sessionOK SessionOK, err error) GetExpectedCSRFToken {
	return func(r *http.Request) (Token, SessionOK, error) {
		return expectedToken, sessionOK, err
	}
}

func mockGetSubmittedCSRFToken(submittedToken Token, err error) GetSubmittedCSRFToken {
	return func(r *http.Request) (Token, error) {
		return submittedToken, err
	}
}

// Test the middleware function
func TestCSRFMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		expectedToken     Token
		submittedToken    Token
		sessionOK         SessionOK
		expectedTokenErr  error
		submittedTokenErr error
		origin            string
		permittedHosts    []string
		isExempt          bool
		expectedStatus    int
	}{
		{
			name:           "GET request, no CSRF check",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "HEAD request, no CSRF check",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS request, no CSRF check",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request, no origin",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST request, origin not permitted",
			method:         http.MethodPost,
			origin:         "http://evil.com",
			permittedHosts: []string{"example.com"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST request, exempt from CSRF check",
			method:         http.MethodPost,
			origin:         "http://example.com",
			isExempt:       true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request, session not OK",
			method:         http.MethodPost,
			origin:         "http://example.com",
			sessionOK:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:             "POST request, error fetching expected token",
			method:           http.MethodPost,
			origin:           "http://example.com",
			sessionOK:        true,
			expectedTokenErr: errors.New("error"),
			expectedStatus:   http.StatusInternalServerError,
		},
		{
			name:           "POST request, expected token empty",
			method:         http.MethodPost,
			origin:         "http://example.com",
			sessionOK:      true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:              "POST request, error fetching submitted token",
			method:            http.MethodPost,
			origin:            "http://example.com",
			sessionOK:         true,
			expectedToken:     "expectedToken",
			submittedTokenErr: errors.New("error"),
			expectedStatus:    http.StatusInternalServerError,
		},
		{
			name:           "POST request, submitted token missing",
			method:         http.MethodPost,
			origin:         "http://example.com",
			sessionOK:      true,
			expectedToken:  "expectedToken",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST request, token mismatch",
			method:         http.MethodPost,
			origin:         "http://example.com",
			sessionOK:      true,
			expectedToken:  "expectedToken",
			submittedToken: "wrongToken",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST request, token match",
			method:         http.MethodPost,
			origin:         "http://example.com",
			sessionOK:      true,
			expectedToken:  "expectedToken",
			submittedToken: "expectedToken",
			expectedStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create the middleware
			middleware := NewMiddleware(Opts{
				GetExpectedCSRFToken:  mockGetExpectedCSRFToken(test.expectedToken, test.sessionOK, test.expectedTokenErr),
				GetSubmittedCSRFToken: mockGetSubmittedCSRFToken(test.submittedToken, test.submittedTokenErr),
				GetIsExempt:           func(r *http.Request) bool { return test.isExempt },
				PermittedHosts:        test.permittedHosts,
			})

			// Create a mock handler to simulate the next handler in the chain
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create a test request and response recorder
			req := httptest.NewRequest(test.method, "http://example.com", nil)
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			rr := httptest.NewRecorder()

			// Apply the middleware
			middleware(nextHandler).ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != test.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, test.expectedStatus)
			}
		})
	}
}

// Test the getLowercaseHost function
func TestGetLowercaseHost(t *testing.T) {
	tests := []struct {
		name         string
		origin       string
		referer      string
		expectedHost string
		expectError  bool
	}{
		{
			name:         "Valid Origin",
			origin:       "http://EXAMPLE.com",
			expectedHost: "example.com",
		},
		{
			name:         "Valid Referer",
			referer:      "https://TEST.com/path",
			expectedHost: "test.com",
		},
		{
			name:         "No Origin or Referer",
			expectedHost: "",
		},
		{
			name:        "Invalid Origin - not a URL",
			origin:      "not-a-url",
			expectError: true,
		},
		{
			name:        "Invalid Origin - missing scheme",
			origin:      "example.com",
			expectError: true,
		},
		{
			name:        "Invalid Origin - missing host",
			origin:      "http://",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.referer != "" {
				req.Header.Set("Referer", test.referer)
			}

			host, err := getLowercaseHost(req)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if host != test.expectedHost {
					t.Errorf("Expected host %q, but got %q", test.expectedHost, host)
				}
			}
		})
	}
}

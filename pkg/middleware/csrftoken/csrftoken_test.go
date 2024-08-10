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
			name:           "POST request, session not OK",
			method:         http.MethodPost,
			sessionOK:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:             "POST request, error fetching expected token",
			method:           http.MethodPost,
			sessionOK:        true,
			expectedTokenErr: errors.New("error"),
			expectedStatus:   http.StatusInternalServerError,
		},
		{
			name:           "POST request, expected token empty",
			method:         http.MethodPost,
			sessionOK:      true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:              "POST request, error fetching submitted token",
			method:            http.MethodPost,
			sessionOK:         true,
			expectedToken:     "expectedToken",
			submittedTokenErr: errors.New("error"),
			expectedStatus:    http.StatusInternalServerError,
		},
		{
			name:           "POST request, submitted token missing",
			method:         http.MethodPost,
			sessionOK:      true,
			expectedToken:  "expectedToken",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST request, token mismatch",
			method:         http.MethodPost,
			sessionOK:      true,
			expectedToken:  "expectedToken",
			submittedToken: "wrongToken",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST request, token match",
			method:         http.MethodPost,
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
			})

			// Create a mock handler to simulate the next handler in the chain
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create a test request and response recorder
			req := httptest.NewRequest(test.method, "http://example.com", nil)
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

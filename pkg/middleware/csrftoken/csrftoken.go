package csrftoken

import (
	"net/http"

	"github.com/sjc5/kit/pkg/response"
)

type SessionOK = bool
type Token = string

type GetExpectedCSRFToken func(r *http.Request) (Token, SessionOK, error)
type GetSubmittedCSRFToken func(r *http.Request) (Token, error)

type Opts struct {
	GetExpectedCSRFToken  GetExpectedCSRFToken
	GetSubmittedCSRFToken GetSubmittedCSRFToken
	GetIsExempt           func(r *http.Request) bool
}

func NewMiddleware(opts Opts) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := response.New(w)

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			if opts.GetIsExempt(r) {
				next.ServeHTTP(w, r)
				return
			}

			expectedToken, sessionOK, err := opts.GetExpectedCSRFToken(r)
			if !sessionOK {
				res.Unauthorized("")
				return
			}
			if err != nil || expectedToken == "" {
				res.InternalServerError("")
				return
			}

			submittedToken, err := opts.GetSubmittedCSRFToken(r)
			if err != nil {
				res.InternalServerError("")
				return
			}
			if submittedToken == "" {
				res.BadRequest("CSRF token missing")
				return
			}

			if submittedToken != expectedToken {
				res.Forbidden("CSRF token mismatch")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

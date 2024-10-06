package csrftoken

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/sjc5/kit/pkg/response"
)

type SessionOK = bool
type Token = string

type GetExpectedCSRFToken func(r *http.Request) (Token, SessionOK, error)
type GetSubmittedCSRFToken func(r *http.Request) (Token, error)

type Opts struct {
	GetExpectedCSRFToken  GetExpectedCSRFToken
	GetSubmittedCSRFToken GetSubmittedCSRFToken
	GetIsExempt           func(r *http.Request) bool // Exempts from CSRF token check (not host check)
	PermittedHosts        []string                   // If len == 0, all hosts are permitted
}

func NewMiddleware(opts Opts) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := response.New(w)

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				next.ServeHTTP(w, r)
				return
			}

			lowercaseHost, err := getLowercaseHost(r)
			if err != nil {
				res.InternalServerError("")
				return
			}
			if lowercaseHost == "" {
				res.BadRequest("Origin not provided")
				return
			}
			if len(opts.PermittedHosts) > 0 {
				permitted := false
				for _, permittedHost := range opts.PermittedHosts {
					if lowercaseHost == strings.ToLower(permittedHost) {
						permitted = true
						break
					}
				}
				if !permitted {
					res.Forbidden("Origin not permitted")
					return
				}
			}

			if opts.GetIsExempt != nil && opts.GetIsExempt(r) {
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

func getLowercaseHost(r *http.Request) (string, error) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return "", nil
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	if originURL.Scheme == "" || originURL.Host == "" {
		return "", errors.New("invalid URL: missing scheme or host")
	}
	return strings.ToLower(originURL.Host), nil
}

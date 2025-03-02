package healthcheck

import (
	"net/http"
	"strings"

	"github.com/sjc5/kit/pkg/response"
)

type Middleware func(http.Handler) http.Handler

// OK returns a middleware that responds with an HTTP 200 OK status code and the
// string "OK" in the response body for GET and HEAD requests to the given endpoint.
func OK(endpoint string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAppropriateMethod := r.Method == http.MethodGet || r.Method == http.MethodHead
			if isAppropriateMethod && strings.EqualFold(r.URL.Path, endpoint) {
				res := response.New(w)
				res.OKText()
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Healthz is a middleware that responds with an HTTP 200 OK status code and the
// string "OK" in the response body for GET and HEAD requests to the "/healthz" endpoint.
var Healthz = OK("/healthz")

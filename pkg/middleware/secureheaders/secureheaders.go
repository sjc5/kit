package secureheaders

import "net/http"

// Sets various security-related headers to responses.
func Middleware(next http.Handler) http.Handler {
	securityHeadersMap := map[string]string{
		"Content-Security-Policy":           "default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests",
		"Cross-Origin-Opener-Policy":        "same-origin",
		"Cross-Origin-Resource-Policy":      "same-origin",
		"Origin-Agent-Cluster":              "?1",
		"Referrer-Policy":                   "no-referrer",
		"Strict-Transport-Security":         "max-age=15552000; includeSubDomains",
		"X-Content-Type-Options":            "nosniff",
		"X-DNS-Prefetch-Control":            "off",
		"X-Download-Options":                "noopen",
		"X-Frame-Options":                   "SAMEORIGIN",
		"X-Permitted-Cross-Domain-Policies": "none",
		"X-XSS-Protection":                  "0",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note: These are all set to the defaults used by helmetjs
		// See https://github.com/helmetjs/helmet?tab=readme-ov-file#reference
		for header, value := range securityHeadersMap {
			w.Header().Set(header, value)
		}
		w.Header().Del("X-Powered-By")
		next.ServeHTTP(w, r)
	})
}

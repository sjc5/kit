package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseSegments(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"empty path", "", nil},
		{"root path", "/", []string{}},
		{"simple path", "/users", []string{"users"}},
		{"multi-segment path", "/api/v1/users", []string{"api", "v1", "users"}},
		{"trailing slash", "/users/", []string{"users"}},
		{"path with parameters", "/users/$id/posts", []string{"users", "$id", "posts"}},
		{"path with splat", "/files/$", []string{"files", "$"}},
		{"multiple slashes", "//api///users", []string{"api", "users"}},
		{"complex path", "/api/v1/users/$user_id/posts/$post_id/comments", []string{"api", "v1", "users", "$user_id", "posts", "$post_id", "comments"}},
		{"unicode path", "/café/über/resumé", []string{"café", "über", "resumé"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSegments(tt.path)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseSegments(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatchCore(t *testing.T) {
	tests := []struct {
		name            string
		patternSegments []string
		realSegments    []string
		wantMatch       bool
		wantParams      Params
		wantScore       int
	}{
		{
			name:            "exact match",
			patternSegments: []string{"users", "profile"},
			realSegments:    []string{"users", "profile"},
			wantMatch:       true,
			wantParams:      Params{},
			wantScore:       6, // 3 + 3
		},
		{
			name:            "parameter match",
			patternSegments: []string{"users", "$id"},
			realSegments:    []string{"users", "123"},
			wantMatch:       true,
			wantParams:      Params{"id": "123"},
			wantScore:       5, // 3 + 2
		},
		{
			name:            "multiple parameters",
			patternSegments: []string{"api", "$version", "users", "$id"},
			realSegments:    []string{"api", "v1", "users", "abc123"},
			wantMatch:       true,
			wantParams:      Params{"version": "v1", "id": "abc123"},
			wantScore:       10, // 3 + 2 + 3 + 2
		},
		{
			name:            "splat parameter",
			patternSegments: []string{"files", "$"},
			realSegments:    []string{"files", "documents", "report.pdf"},
			wantMatch:       true,
			wantParams:      Params{},
			wantScore:       4, // 3 + 1
		},
		{
			name:            "no match - different segments",
			patternSegments: []string{"users", "profile"},
			realSegments:    []string{"users", "settings"},
			wantMatch:       false,
		},
		{
			name:            "no match - pattern longer than path",
			patternSegments: []string{"users", "profile", "settings"},
			realSegments:    []string{"users", "profile"},
			wantMatch:       false,
		},
		{
			name:            "partial match",
			patternSegments: []string{"users"},
			realSegments:    []string{"users", "profile"},
			wantMatch:       true,
			wantParams:      Params{},
			wantScore:       3,
		},
		{
			name:            "mixed segment types",
			patternSegments: []string{"api", "$version", "users", "$id", "$"},
			realSegments:    []string{"api", "v2", "users", "123", "profile", "avatar"},
			wantMatch:       true,
			wantParams:      Params{"version": "v2", "id": "123"},
			wantScore:       11, // 3 + 2 + 3 + 2 + 1
		},
		{
			name:            "empty segments",
			patternSegments: []string{},
			realSegments:    []string{"users", "123"},
			wantMatch:       true,
			wantParams:      Params{},
			wantScore:       0,
		},
		{
			name:            "unicode segments",
			patternSegments: []string{"café", "$id"},
			realSegments:    []string{"café", "über"},
			wantMatch:       true,
			wantParams:      Params{"id": "über"},
			wantScore:       5, // 3 + 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := MatchCore(tt.patternSegments, tt.realSegments)

			if ok != tt.wantMatch {
				t.Errorf("MatchCore() match = %v, want %v", ok, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				if result != nil {
					t.Errorf("MatchCore() match = false but result = %v, want nil", result)
				}
				return
			}

			if !reflect.DeepEqual(result.Params, tt.wantParams) {
				t.Errorf("MatchCore() params = %v, want %v", result.Params, tt.wantParams)
			}

			if result.Score != tt.wantScore {
				t.Errorf("MatchCore() score = %d, want %d", result.Score, tt.wantScore)
			}
		})
	}
}

func TestRouter_AddRoute(t *testing.T) {
	patterns := []string{
		"/",
		"/users",
		"/users/$id",
		"/api/v1/resources/$resource",
		"/files/$",
	}

	router := NewRouter()
	for _, pattern := range patterns {
		router.AddRoute(pattern)
	}

	if len(router.routes) != len(patterns) {
		t.Errorf("AddRoute() created %d routes, want %d", len(router.routes), len(patterns))
	}

	for _, pattern := range patterns {
		route, exists := router.routes[pattern]
		if !exists {
			t.Errorf("AddRoute() did not store route for pattern %q", pattern)
			continue
		}

		expectedSegments := ParseSegments(pattern)
		if !reflect.DeepEqual(route.Segments, expectedSegments) {
			t.Errorf("AddRoute() for pattern %q created segments %v, want %v",
				pattern, route.Segments, expectedSegments)
		}
	}

	// Test overwriting existing route
	router.AddRoute("/users")
	if len(router.routes) != len(patterns) {
		t.Errorf("AddRoute() with existing pattern modified route count, got %d want %d",
			len(router.routes), len(patterns))
	}
}

func TestRouter_FindBestMatch(t *testing.T) {
	tests := []struct {
		name         string
		routes       []string
		path         string
		wantMatch    bool
		wantPattern  string
		wantParams   Params
		preCondition MatchPreConditionChecker
	}{
		{
			name:        "exact match",
			routes:      []string{"/", "/users", "/posts"},
			path:        "/users",
			wantMatch:   true,
			wantPattern: "/users",
			wantParams:  Params{},
		},
		{
			name:        "parameter match",
			routes:      []string{"/users", "/users/$id", "/users/profile"},
			path:        "/users/123",
			wantMatch:   true,
			wantPattern: "/users/$id",
			wantParams:  Params{"id": "123"},
		},
		{
			name:        "multiple matches - select best score",
			routes:      []string{"/", "/api", "/api/$version", "/api/v1"},
			path:        "/api/v1",
			wantMatch:   true,
			wantPattern: "/api/v1", // Exact match has higher score than parameter match
			wantParams:  Params{},
		},
		{
			name:        "splat match",
			routes:      []string{"/files", "/files/$"},
			path:        "/files/documents/report.pdf",
			wantMatch:   true,
			wantPattern: "/files/$",
			wantParams:  Params{},
		},
		{
			name:      "no match",
			routes:    []string{"/users", "/posts", "/settings"},
			path:      "/profile",
			wantMatch: false,
		},
		{
			name: "complex nested paths",
			routes: []string{
				"/api/v1/users",
				"/api/$version/users",
				"/api/v1/users/$id",
				"/api/$version/users/$id",
				"/api/v1/users/$id/posts",
				"/api/$version/users/$id/posts",
			},
			path:        "/api/v2/users/123/posts",
			wantMatch:   true,
			wantPattern: "/api/$version/users/$id/posts",
			wantParams:  Params{"version": "v2", "id": "123"},
		},
		{
			name:        "with pre-condition - match",
			routes:      []string{"/api/v1/users", "/api/v2/users"},
			path:        "/api/v1/users",
			wantMatch:   true,
			wantPattern: "/api/v1/users",
			wantParams:  Params{},
			preCondition: func(r *http.Request, route *Route) bool {
				// Only match v1 API
				return len(route.Segments) >= 2 && route.Segments[1] == "v1"
			},
		},
		{
			name:      "with pre-condition - no match",
			routes:    []string{"/api/v1/users", "/api/v2/users"},
			path:      "/api/v1/users",
			wantMatch: false,
			preCondition: func(r *http.Request, route *Route) bool {
				// Only match v2 API
				return len(route.Segments) >= 2 && route.Segments[1] == "v2"
			},
		},
		{
			name:      "empty routes",
			routes:    []string{},
			path:      "/users",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			for _, pattern := range tt.routes {
				router.AddRoute(pattern)
			}

			if tt.preCondition != nil {
				router.SetMatchPreConditionChecker(tt.preCondition)
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			match, ok := router.FindBestMatch(req)

			if ok != tt.wantMatch {
				t.Errorf("FindBestMatch() match = %v, want %v", ok, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				if match != nil {
					t.Errorf("FindBestMatch() match = %v, want nil", match)
				}
				return
			}

			if match.Pattern != tt.wantPattern {
				t.Errorf("FindBestMatch() pattern = %q, want %q", match.Pattern, tt.wantPattern)
			}

			if !reflect.DeepEqual(match.Params, tt.wantParams) {
				t.Errorf("FindBestMatch() params = %v, want %v", match.Params, tt.wantParams)
			}
		})
	}
}

func TestRouter_PreConditionChecker(t *testing.T) {
	router := NewRouter()
	router.AddRoute("/users")
	router.AddRoute("/posts")
	router.AddRoute("/api/v1/users")
	router.AddRoute("/api/v2/users")

	tests := []struct {
		name        string
		method      string
		path        string
		checker     MatchPreConditionChecker
		wantMatch   bool
		wantPattern string
	}{
		{
			name:   "match GET requests only - GET request",
			method: http.MethodGet,
			path:   "/users",
			checker: func(r *http.Request, route *Route) bool {
				return r.Method == http.MethodGet
			},
			wantMatch:   true,
			wantPattern: "/users",
		},
		{
			name:   "match GET requests only - POST request",
			method: http.MethodPost,
			path:   "/users",
			checker: func(r *http.Request, route *Route) bool {
				return r.Method == http.MethodGet
			},
			wantMatch: false,
		},
		{
			name:   "match v1 API only",
			method: http.MethodGet,
			path:   "/api/v1/users",
			checker: func(r *http.Request, route *Route) bool {
				return len(route.Segments) >= 2 && route.Segments[1] == "v1"
			},
			wantMatch:   true,
			wantPattern: "/api/v1/users",
		},
		{
			name:   "match v1 API only - v2 request",
			method: http.MethodGet,
			path:   "/api/v2/users",
			checker: func(r *http.Request, route *Route) bool {
				return len(route.Segments) >= 2 && route.Segments[1] == "v1"
			},
			wantMatch: false,
		},
		{
			name:   "complex condition - path with auth header",
			method: http.MethodGet,
			path:   "/api/v2/users",
			checker: func(r *http.Request, route *Route) bool {
				// Only match API v2 routes with Authorization header
				isV2 := len(route.Segments) >= 2 && route.Segments[1] == "v2"
				hasAuth := r.Header.Get("Authorization") != ""
				return isV2 && hasAuth
			},
			wantMatch: false, // No auth header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.SetMatchPreConditionChecker(tt.checker)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			match, ok := router.FindBestMatch(req)

			if ok != tt.wantMatch {
				t.Errorf("FindBestMatch() with PreConditionChecker: match = %v, want %v", ok, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				return
			}

			if match.Pattern != tt.wantPattern {
				t.Errorf("FindBestMatch() with PreConditionChecker: pattern = %q, want %q",
					match.Pattern, tt.wantPattern)
			}
		})
	}

	// Test with nil checker
	t.Run("nil checker", func(t *testing.T) {
		router.SetMatchPreConditionChecker(nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
		match, ok := router.FindBestMatch(req)

		if !ok {
			t.Errorf("FindBestMatch() with nil checker: match = false, want true")
			return
		}

		if match.Pattern != "/api/v2/users" {
			t.Errorf("FindBestMatch() with nil checker: pattern = %q, want %q",
				match.Pattern, "/api/v2/users")
		}
	})
}

func BenchmarkParseSegments(b *testing.B) {
	paths := []string{
		"/",
		"/users",
		"/api/v1/users",
		"/api/v1/users/123/posts/456/comments",
		"/files/documents/reports/quarterly/q3-2023.pdf",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		_ = ParseSegments(path)
	}
}

func BenchmarkMatchCore(b *testing.B) {
	patterns := [][]string{
		{"users"},
		{"api", "v1", "users"},
		{"api", "$version", "users", "$id"},
		{"files", "$"},
	}

	paths := [][]string{
		{"users"},
		{"api", "v1", "users"},
		{"api", "v2", "users", "123"},
		{"files", "documents", "report.pdf"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		patternIdx := i % len(patterns)
		pathIdx := i % len(paths)
		_, _ = MatchCore(patterns[patternIdx], paths[pathIdx])
	}
}

func BenchmarkRouter_FindBestMatch(b *testing.B) {
	router := NewRouter()
	router.AddRoute("/")
	router.AddRoute("/users")
	router.AddRoute("/users/$id")
	router.AddRoute("/users/$id/profile")
	router.AddRoute("/api/v1/users")
	router.AddRoute("/api/$version/users")
	router.AddRoute("/api/v1/users/$id")
	router.AddRoute("/api/$version/users/$id")
	router.AddRoute("/files/$")

	paths := []string{
		"/",
		"/users",
		"/users/123",
		"/users/123/profile",
		"/api/v1/users",
		"/api/v2/users",
		"/api/v1/users/456",
		"/files/documents/report.pdf",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		req := httptest.NewRequest(http.MethodGet, path, nil)
		_, _ = router.FindBestMatch(req)
	}
}

package router

import (
	"reflect"
	"testing"
)

func TestRouter_FindBestMatch(t *testing.T) {
	tests := []struct {
		name              string
		routes            []string
		path              string
		wantPattern       string
		wantParams        Params
		wantSplatSegments []string
	}{
		// index
		{
			name:        "root path",
			routes:      []string{"/", "/$"},
			path:        "/",
			wantPattern: "/",
			wantParams:  nil,
		},
		{
			name:              "exact match",
			routes:            []string{"/", "/users", "/posts"},
			path:              "/users",
			wantPattern:       "/users",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "parameter match",
			routes:            []string{"/users", "/users/$id", "/users/profile"},
			path:              "/users/123",
			wantPattern:       "/users/$id",
			wantParams:        Params{"id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "multiple matches",
			routes:            []string{"/", "/api", "/api/$version", "/api/v1"},
			path:              "/api/v1",
			wantPattern:       "/api/v1",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "splat match",
			routes:            []string{"/files", "/files/$"},
			path:              "/files/documents/report.pdf",
			wantPattern:       "/files/$",
			wantParams:        nil,
			wantSplatSegments: []string{"documents", "report.pdf"},
		},
		{
			name:              "no match",
			routes:            []string{"/users", "/posts", "/settings"},
			path:              "/profile",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
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
			path:              "/api/v2/users/123/posts",
			wantPattern:       "/api/$version/users/$id/posts",
			wantParams:        Params{"version": "v2", "id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "empty routes",
			routes:            []string{},
			path:              "/users",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "many params",
			routes:            []string{"/api/$p1/$p2/$p3/$p4/$p5"},
			path:              "/api/a/b/c/d/e",
			wantPattern:       "/api/$p1/$p2/$p3/$p4/$p5",
			wantParams:        Params{"p1": "a", "p2": "b", "p3": "c", "p4": "d", "p5": "e"},
			wantSplatSegments: nil,
		},
		{
			name:              "nested no match",
			routes:            []string{"/users/$id", "/users/$id/profile"},
			path:              "users/123/settings",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := Router{}
			for _, pattern := range tt.routes {
				router.AddRoute(pattern)
			}

			match, _ := router.FindBestMatch(tt.path)

			wantMatch := tt.wantPattern != ""

			if wantMatch && match == nil {
				t.Errorf("FindBestMatch() match for %s = nil -- want %s", tt.path, tt.wantPattern)
				return
			}

			if !wantMatch {
				if match != nil {
					t.Errorf("FindBestMatch() match for %s = %v -- want nil", tt.path, match.RegisteredRoute.Pattern)
				}
				return
			}

			if match.Pattern != tt.wantPattern {
				t.Errorf("FindBestMatch() pattern = %q, want %q", match.Pattern, tt.wantPattern)
			}

			// Compare params, allowing nil == empty map
			if tt.wantParams == nil && len(match.Params) > 0 {
				t.Errorf("FindBestMatch() params = %v, want nil", match.Params)
			} else if tt.wantParams != nil && !reflect.DeepEqual(match.Params, tt.wantParams) {
				t.Errorf("FindBestMatch() params = %v, want %v", match.Params, tt.wantParams)
			}

			// Compare splat segments
			if !reflect.DeepEqual(match.SplatValues, tt.wantSplatSegments) {
				t.Errorf("FindBestMatch() splat segments = %v, want %v", match.SplatValues, tt.wantSplatSegments)
			}
		})
	}
}

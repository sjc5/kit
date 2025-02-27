package matcher

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

func TestFindBestMatch(t *testing.T) {
	tests := []struct {
		name              string
		patterns          []string
		path              string
		wantPattern       string
		wantParams        Params
		wantSplatSegments []string
	}{
		// index
		{
			name:        "root path",
			patterns:    []string{"/", "/*"},
			path:        "/",
			wantPattern: "/",
			wantParams:  nil,
		},
		{
			name:              "exact match",
			patterns:          []string{"/", "/users", "/posts"},
			path:              "/users",
			wantPattern:       "/users",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "parameter match",
			patterns:          []string{"/users", "/users/:id", "/users/profile"},
			path:              "/users/123",
			wantPattern:       "/users/:id",
			wantParams:        Params{"id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "multiple matches",
			patterns:          []string{"/", "/api", "/api/:version", "/api/v1"},
			path:              "/api/v1",
			wantPattern:       "/api/v1",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "splat match",
			patterns:          []string{"/files", "/files/*"},
			path:              "/files/documents/report.pdf",
			wantPattern:       "/files/*",
			wantParams:        nil,
			wantSplatSegments: []string{"documents", "report.pdf"},
		},
		{
			name:              "no match",
			patterns:          []string{"/users", "/posts", "/settings"},
			path:              "/profile",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name: "complex nested paths",
			patterns: []string{
				"/api/v1/users",
				"/api/:version/users",
				"/api/v1/users/:id",
				"/api/:version/users/:id",
				"/api/v1/users/:id/posts",
				"/api/:version/users/:id/posts",
			},
			path:              "/api/v2/users/123/posts",
			wantPattern:       "/api/:version/users/:id/posts",
			wantParams:        Params{"version": "v2", "id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "no patterns",
			patterns:          []string{},
			path:              "/users",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "many params",
			patterns:          []string{"/api/:p1/:p2/:p3/:p4/:p5"},
			path:              "/api/a/b/c/d/e",
			wantPattern:       "/api/:p1/:p2/:p3/:p4/:p5",
			wantParams:        Params{"p1": "a", "p2": "b", "p3": "c", "p4": "d", "p5": "e"},
			wantSplatSegments: nil,
		},
		{
			name:              "nested no match",
			patterns:          []string{"/users/:id", "/users/:id/profile"},
			path:              "users/123/settings",
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil)
			for _, pattern := range tt.patterns {
				m.RegisterPattern(pattern)
			}

			match, _ := m.FindBestMatch(tt.path)

			wantMatch := tt.wantPattern != ""

			if wantMatch && match == nil {
				t.Errorf("FindBestMatch() match for %s = nil -- want %s", tt.path, tt.wantPattern)
				return
			}

			if !wantMatch {
				if match != nil {
					t.Errorf("FindBestMatch() match for %s = %v -- want nil", tt.path, match.RegisteredPattern.pattern)
				}
				return
			}

			if match.pattern != tt.wantPattern {
				t.Errorf("FindBestMatch() pattern = %q, want %q", match.pattern, tt.wantPattern)
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

func setupNonNestedMatcherForBenchmark(scale string) *Matcher {
	m := New(nil)

	switch scale {
	case "small":
		// Basic patterns for simple tests
		m.RegisterPattern("/")
		m.RegisterPattern("/users")
		m.RegisterPattern("/users/:id")
		m.RegisterPattern("/users/:id/profile")
		m.RegisterPattern("/api/v1/users")
		m.RegisterPattern("/api/:version/users")
		m.RegisterPattern("/api/v1/users/:id")
		m.RegisterPattern("/files/*")

	case "medium":
		// RESTful API-style patterns
		for i := range 1_000 {
			m.RegisterPattern(fmt.Sprintf("/api/v%d/users", i%5))
			m.RegisterPattern(fmt.Sprintf("/api/v%d/users/:id", i%5))
			m.RegisterPattern(fmt.Sprintf("/api/v%d/users/:id/posts/:post_id", i%5))
			m.RegisterPattern(fmt.Sprintf("/files/bucket%d/*", i%10))
		}

	case "large":
		// Complex application patterns
		for i := range 10_000 {
			// Static patterns
			m.RegisterPattern(fmt.Sprintf("/api/v%d/users", i%10))
			m.RegisterPattern(fmt.Sprintf("/api/v%d/products", i%10))
			m.RegisterPattern(fmt.Sprintf("/docs/section%d", i%100))

			// Dynamic patterns
			m.RegisterPattern(fmt.Sprintf("/api/v%d/users/:id/posts/:post_id", i%10))
			m.RegisterPattern(fmt.Sprintf("/api/v%d/products/:category/:id", i%10))

			// Splat patterns
			m.RegisterPattern(fmt.Sprintf("/files/bucket%d/*", i%20))
		}
	}

	return m
}

// generateNonNestedPathsForBenchmark creates test paths for different scenarios
func generateNonNestedPathsForBenchmark(scale string) []string {
	switch scale {
	case "small":
		return []string{
			"/",
			"/users",
			"/users/123",
			"/users/123/profile",
			"/api/v1/users",
			"/api/v2/users",
			"/files/document.pdf",
		}
	case "medium", "large":
		paths := make([]string, 0, 1000)

		// Static paths (40%)
		for i := range 400 {
			paths = append(paths, fmt.Sprintf("/api/v%d/users", i%5))
		}

		// Dynamic paths (40%)
		for i := range 400 {
			paths = append(paths, fmt.Sprintf("/api/v%d/users/%d/posts/%d", i%5, i, i%100))
		}

		// Splat paths (20%)
		for i := range 200 {
			paths = append(paths, fmt.Sprintf("/files/bucket%d/path/to/file%d.txt", i%10, i))
		}

		return paths
	}
	return nil
}

func BenchmarkFindBestMatchSimple(b *testing.B) {
	scenarios := []struct {
		name string
		path string
	}{
		{"StaticPattern", "/api/v1/users"},
		{"DynamicPattern", "/api/v1/users/123/posts/456"},
		{"SplatPattern", "/files/bucket1/deep/path/file.txt"},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			var memStatsBefore runtime.MemStats
			runtime.ReadMemStats(&memStatsBefore)

			m := setupNonNestedMatcherForBenchmark("medium")

			runtime.GC()
			var memStatsAfter runtime.MemStats
			runtime.ReadMemStats(&memStatsAfter)

			b.ResetTimer()
			b.ReportAllocs()

			matches := 0
			for b.Loop() {
				match, ok := m.FindBestMatch(s.path)
				if ok {
					matches++
				}
				runtime.KeepAlive(match)
			}
		})
	}
}

func BenchmarkFindBestMatchAtScale(b *testing.B) {
	scales := []string{"small", "medium", "large"}

	for _, scale := range scales {
		b.Run(fmt.Sprintf("Scale_%s", scale), func(b *testing.B) {
			m := setupNonNestedMatcherForBenchmark(scale)
			paths := generateNonNestedPathsForBenchmark(scale)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				path := paths[i%len(paths)]
				match, _ := m.FindBestMatch(path)
				runtime.KeepAlive(match)
			}
		})
	}

	b.Run("WorstCase_DeepNested", func(b *testing.B) {
		m := setupNonNestedMatcherForBenchmark("large")
		path := "/api/v9/users/999/posts/999"
		b.ReportAllocs()
		for b.Loop() {
			match, _ := m.FindBestMatch(path)
			runtime.KeepAlive(match)
		}
	})
}

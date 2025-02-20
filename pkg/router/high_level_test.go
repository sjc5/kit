package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
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

func TestRouter_FindBestMatch(t *testing.T) {
	tests := []struct {
		name        string
		routes      []string
		path        string
		wantMatch   bool
		wantPattern string
		wantParams  Params // nil for static routes
	}{
		{
			name:        "exact match",
			routes:      []string{"/", "/users", "/posts"},
			path:        "/users",
			wantMatch:   true,
			wantPattern: "/users",
			wantParams:  nil, // static route, no params needed
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
			wantPattern: "/api/v1",
			wantParams:  nil, // static route wins
		},
		{
			name:        "splat match",
			routes:      []string{"/files", "/files/$"},
			path:        "/files/documents/report.pdf",
			wantMatch:   true,
			wantPattern: "/files/$",
			wantParams:  nil, // splat route doesn't need params
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

			// Compare params, allowing nil == empty map
			if tt.wantParams == nil && len(match.Params) > 0 {
				t.Errorf("FindBestMatch() params = %v, want nil", match.Params)
			} else if tt.wantParams != nil && !reflect.DeepEqual(match.Params, tt.wantParams) {
				t.Errorf("FindBestMatch() params = %v, want %v", match.Params, tt.wantParams)
			}
		})
	}
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

func BenchmarkRouter_LargeScale(b *testing.B) {
	router := NewRouter()

	// Generate 1_000 routes with varied patterns
	for i := 0; i < 1_000; i++ {
		// Add static routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users", i%5))
		router.AddRoute(fmt.Sprintf("/api/v%d/products", i%5))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders", i%5))

		// Add dynamic routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users/$id", i%5))
		router.AddRoute(fmt.Sprintf("/api/v%d/products/$category/$id", i%5))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders/$order_id/items/$item_id", i%5))

		// Add splat routes
		router.AddRoute(fmt.Sprintf("/files/bucket%d/$", i%10))
		router.AddRoute(fmt.Sprintf("/assets/public%d/$", i%10))
	}

	// Add original test routes to ensure backwards compatibility
	router.AddRoute("/")
	router.AddRoute("/users")
	router.AddRoute("/users/$id")
	router.AddRoute("/users/$id/profile")
	router.AddRoute("/api/v1/users")
	router.AddRoute("/api/$version/users")
	router.AddRoute("/api/v1/users/$id")
	router.AddRoute("/api/$version/users/$id")
	router.AddRoute("/files/$")

	// Generate test paths that will match the routes
	paths := make([]string, 0, 1_000)

	// Static paths
	for i := 0; i < 300; i++ {
		paths = append(paths, fmt.Sprintf("/api/v%d/users", i%5))
		paths = append(paths, fmt.Sprintf("/api/v%d/products", i%5))
		paths = append(paths, fmt.Sprintf("/api/v%d/orders", i%5))
	}

	// Dynamic paths
	for i := 0; i < 300; i++ {
		paths = append(paths, fmt.Sprintf("/api/v%d/users/%d", i%5, i))
		paths = append(paths, fmt.Sprintf("/api/v%d/products/category%d/%d", i%5, i%10, i))
		paths = append(paths, fmt.Sprintf("/api/v%d/orders/order%d/items/item%d", i%5, i, i%50))
	}

	// Splat paths
	for i := 0; i < 100; i++ {
		paths = append(paths, fmt.Sprintf("/files/bucket%d/path/to/file%d.txt", i%10, i))
		paths = append(paths, fmt.Sprintf("/assets/public%d/js/app%d.js", i%10, i))
	}

	b.ResetTimer()

	b.Run("All_Routes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := paths[i%len(paths)]
			req := httptest.NewRequest(http.MethodGet, path, nil)
			_, _ = router.FindBestMatch(req)
		}
	})
}

func BenchmarkRouter_ExtremeScale(b *testing.B) {
	router := NewRouter()

	// Generate 10,000+ routes with varied patterns
	for i := 0; i < 10000; i++ {
		// Basic static routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/products", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/customers", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/inventory", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/analytics", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/metrics", i%10))
		router.AddRoute(fmt.Sprintf("/docs/section%d", i%100))
		router.AddRoute(fmt.Sprintf("/blog/post%d", i%500))
		router.AddRoute(fmt.Sprintf("/static/bundle%d", i%50))

		// Complex static routes
		router.AddRoute(fmt.Sprintf("/regions/r%d/zones/z%d/clusters/c%d", i%5, i%10, i%20))
		router.AddRoute(fmt.Sprintf("/departments/d%d/teams/t%d/projects/p%d", i%8, i%12, i%25))

		// Simple dynamic routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users/$id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/products/$id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders/$id", i%10))

		// Medium complexity dynamic routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users/$user_id/posts/$post_id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/products/$category/$id/variants/$variant_id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders/$order_id/items/$item_id/tracking", i%10))

		// Complex dynamic routes
		router.AddRoute(fmt.Sprintf("/api/v%d/organizations/$org_id/teams/$team_id/members/$user_id/roles/$role_id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/projects/$project_id/environments/$env_id/deployments/$deploy_id/stages/$stage_id", i%10))

		// Splat routes at different depths
		router.AddRoute(fmt.Sprintf("/files/bucket%d/$", i%20))
		router.AddRoute(fmt.Sprintf("/assets/public%d/js/$", i%20))
		router.AddRoute(fmt.Sprintf("/logs/system%d/traces/$", i%20))
		router.AddRoute(fmt.Sprintf("/backup/store%d/archives/$", i%20))
	}

	// Generate 10,000 test paths that will match the routes
	paths := make([]string, 0, 10000)

	// Static paths (40% of total)
	for i := 0; i < 4000; i++ {
		paths = append(paths, fmt.Sprintf("/api/v%d/users", i%10))
		paths = append(paths, fmt.Sprintf("/api/v%d/products", i%10))
		paths = append(paths, fmt.Sprintf("/api/v%d/orders", i%10))
		paths = append(paths, fmt.Sprintf("/docs/section%d", i%100))
		paths = append(paths, fmt.Sprintf("/regions/r%d/zones/z%d/clusters/c%d", i%5, i%10, i%20))
	}

	// Simple dynamic paths (20% of total)
	for i := 0; i < 2000; i++ {
		paths = append(paths, fmt.Sprintf("/api/v%d/users/%d", i%10, i))
		paths = append(paths, fmt.Sprintf("/api/v%d/products/%d", i%10, i))
		paths = append(paths, fmt.Sprintf("/api/v%d/orders/%d", i%10, i))
	}

	// Complex dynamic paths (30% of total)
	for i := 0; i < 3000; i++ {
		paths = append(paths, fmt.Sprintf("/api/v%d/users/%d/posts/%d", i%10, i, i%500))
		paths = append(paths, fmt.Sprintf("/api/v%d/products/category%d/%d/variants/%d", i%10, i%20, i, i%10))
		paths = append(paths, fmt.Sprintf("/api/v%d/organizations/org%d/teams/team%d/members/user%d/roles/role%d",
			i%10, i%100, i%50, i%1000, i%5))
	}

	// Splat paths (10% of total)
	for i := 0; i < 1000; i++ {
		paths = append(paths, fmt.Sprintf("/files/bucket%d/really/deep/path/to/file%d.txt", i%20, i))
		paths = append(paths, fmt.Sprintf("/assets/public%d/js/vendor/lib/module%d.js", i%20, i))
		paths = append(paths, fmt.Sprintf("/logs/system%d/traces/2024/02/20/hour%d/trace%d.log", i%20, i%24, i))
	}

	b.ResetTimer()

	b.Run("AllRoutes_10k", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := paths[i%len(paths)]
			req := httptest.NewRequest(http.MethodGet, path, nil)
			_, _ = router.FindBestMatch(req)
		}
	})

	// Also benchmark specific route types separately
	b.Run("StaticRoutes_Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := paths[i%4000] // Only use static paths
			req := httptest.NewRequest(http.MethodGet, path, nil)
			_, _ = router.FindBestMatch(req)
		}
	})

	b.Run("DynamicRoutes_Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := paths[4000+(i%5000)] // Only use dynamic paths
			req := httptest.NewRequest(http.MethodGet, path, nil)
			_, _ = router.FindBestMatch(req)
		}
	})

	b.Run("SplatRoutes_Only", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := paths[9000+(i%1000)] // Only use splat paths
			req := httptest.NewRequest(http.MethodGet, path, nil)
			_, _ = router.FindBestMatch(req)
		}
	})

	b.Run("WorstCase_DeepNested", func(b *testing.B) {
		// Test worst-case scenario with very deep paths
		worstCasePath := "/api/v5/organizations/org999/teams/team99/members/user999/roles/role9"
		req := httptest.NewRequest(http.MethodGet, worstCasePath, nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = router.FindBestMatch(req)
		}
	})
}

func BenchmarkRouter_WithMetrics(b *testing.B) {
	// Memory stats before
	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	router := NewRouter()

	// Generate 10,000+ routes with varied patterns
	for i := 0; i < 10000; i++ {
		// Basic static routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/products", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/orders", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/customers", i%10))

		// Dynamic routes
		router.AddRoute(fmt.Sprintf("/api/v%d/users/$id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/products/$category/$id", i%10))
		router.AddRoute(fmt.Sprintf("/api/v%d/users/$user_id/posts/$post_id", i%10))

		// Splat routes
		router.AddRoute(fmt.Sprintf("/files/bucket%d/$", i%20))
		router.AddRoute(fmt.Sprintf("/assets/public%d/$", i%20))
	}

	// Memory stats after route creation
	runtime.ReadMemStats(&memStatsAfter)
	routerMemory := memStatsAfter.HeapAlloc - memStatsBefore.HeapAlloc

	b.ResetTimer()

	// Run sub-benchmarks with memory metrics
	runRouterBenchmark := func(b *testing.B, path string) {
		b.ReportAllocs()
		var totalAllocs uint64
		var matchCount int

		b.RunParallel(func(pb *testing.PB) {
			var memBefore, memAfter runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			req := httptest.NewRequest(http.MethodGet, path, nil)
			for pb.Next() {
				match, ok := router.FindBestMatch(req)
				if ok {
					matchCount++
				}
				// Prevent compiler from optimizing away the match
				runtime.KeepAlive(match)
			}

			runtime.ReadMemStats(&memAfter)
			totalAllocs += memAfter.TotalAlloc - memBefore.TotalAlloc
		})

		// Report custom metrics
		b.ReportMetric(float64(totalAllocs)/float64(b.N), "allocs/op")
		b.ReportMetric(float64(routerMemory), "router_bytes")
		b.ReportMetric(float64(matchCount), "matches")
	}

	// Test different path types
	b.Run("Static_Route", func(b *testing.B) {
		runRouterBenchmark(b, "/api/v1/users")
	})

	b.Run("Dynamic_Route", func(b *testing.B) {
		runRouterBenchmark(b, "/api/v1/users/123/posts/456")
	})

	b.Run("Splat_Route", func(b *testing.B) {
		runRouterBenchmark(b, "/files/bucket1/deep/path/file.txt")
	})

	b.Run("Mixed_Routes", func(b *testing.B) {
		paths := []string{
			"/api/v1/users",
			"/api/v2/products/category1/123",
			"/files/bucket1/some/path/file.txt",
			"/api/v3/users/456/posts/789",
		}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				path := paths[i%len(paths)]
				req := httptest.NewRequest(http.MethodGet, path, nil)
				match, _ := router.FindBestMatch(req)
				runtime.KeepAlive(match)
				i++
			}
		})
	})

	// Worst case scenario
	b.Run("Worst_Case", func(b *testing.B) {
		runRouterBenchmark(b, "/api/v9/users/999/posts/999")
	})

	// Profile memory for router structure
	b.Run("Router_Size", func(b *testing.B) {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		b.ReportMetric(float64(m.HeapAlloc), "heap_bytes")
		b.ReportMetric(float64(m.HeapObjects), "heap_objects")
	})
}

// Helper benchmark to measure parameter map allocations
func BenchmarkParamsAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		params := make(Params)
		params["id"] = "123"
		runtime.KeepAlive(params)
	}
}

// Helper benchmark to measure ParseSegments memory usage
func BenchmarkParseSegmentsMemory(b *testing.B) {
	paths := []string{
		"/api/v1/users",
		"/api/v1/users/123/posts/456",
		"/files/bucket1/deep/path/file.txt",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			segments := ParseSegments(paths[i%len(paths)])
			runtime.KeepAlive(segments)
			i++
		}
	})
}

func TestRouter_ManyParams(t *testing.T) {
	router := NewRouter()
	router.AddRoute("/api/$p1/$p2/$p3/$p4/$p5")

	req := httptest.NewRequest(http.MethodGet, "/api/a/b/c/d/e", nil)
	match, ok := router.FindBestMatch(req)

	if !ok {
		t.Fatal("Expected a match")
	}
	expected := Params{
		"p1": "a",
		"p2": "b",
		"p3": "c",
		"p4": "d",
		"p5": "e",
	}
	if !reflect.DeepEqual(match.Params, expected) {
		t.Errorf("Params = %v, want %v", match.Params, expected)
	}
}

func TestRouter_NestedNoMatch(t *testing.T) {
	router := NewRouter()
	router.AddRoute("/users/$id")
	router.AddRoute("/users/$id/profile")

	req := httptest.NewRequest(http.MethodGet, "/users/123/settings", nil)
	match, ok := router.FindBestMatch(req)

	if ok {
		t.Errorf("Expected no match, got %v", match)
	}
	if match != nil {
		t.Errorf("Expected nil match, got %v", match)
	}
}

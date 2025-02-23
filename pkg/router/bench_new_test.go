package router

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// Compare path operations
func BenchmarkPathOperations(b *testing.B) {
	paths := []string{
		"/",
		"/api/v1/users",
		"/api/v1/users/123/posts/456/comments",
		"/files/documents/reports/quarterly/q3-2023.pdf",
	}

	b.Run("ParseSegments", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			path := paths[i%len(paths)]
			segments := ParseSegments(path)
			runtime.KeepAlive(segments)
		}
	})
}

// setupNestedRouter creates a router with realistic nested routes
func setupNestedRouter() *Router {
	router := &Router{}
	router.NestedIndexSignifier = "_index"
	router.ShouldExcludeSegmentFunc = func(segment string) bool {
		return strings.HasPrefix(segment, "__")
	}

	// Add all the nested patterns from the test suite
	patterns := []string{
		"/_index",
		"/articles/_index",
		"/articles/test/articles/_index",
		"/bear/_index",
		"/dashboard/_index",
		"/dashboard/customers/_index",
		"/dashboard/customers/$customer_id/_index",
		"/dashboard/customers/$customer_id/orders/_index",
		"/dynamic-index/$pagename/_index",
		"/lion/_index",
		"/tiger/_index",
		"/tiger/$tiger_id/_index",
		"/$",
		"/bear",
		"/bear/$bear_id",
		"/bear/$bear_id/$",
		"/dashboard",
		"/dashboard/$",
		"/dashboard/customers",
		"/dashboard/customers/$customer_id",
		"/dashboard/customers/$customer_id/orders",
		"/dashboard/customers/$customer_id/orders/$order_id",
		"/dynamic-index/index",
		"/lion",
		"/lion/$",
		"/tiger",
		"/tiger/$tiger_id",
		"/tiger/$tiger_id/$tiger_cub_id",
		"/tiger/$tiger_id/$",
	}

	for _, pattern := range patterns {
		router.AddRoute(pattern)
	}
	return router
}

// generateNestedTestPaths creates test paths that exercise different routing scenarios
func generateNestedTestPaths() []string {
	return []string{
		"/",                                   // Root index
		"/dashboard",                          // Static route with index
		"/dashboard/customers",                // Nested static route
		"/dashboard/customers/123",            // Route with params
		"/dashboard/customers/123/orders",     // Deep nested route
		"/dashboard/customers/123/orders/456", // Deep nested route with multiple params
		"/tiger",                              // Another static route
		"/tiger/123",                          // Dynamic route
		"/tiger/123/456",                      // Dynamic route with multiple params
		"/tiger/123/456/789",                  // Route with splat
		"/bear/123/456/789",                   // Different route with splat
		"/articles/test/articles",             // Deeply nested static route
		"/does-not-exist",                     // Non-existent route (tests splat handling)
		"/dashboard/unknown/path",             // Tests dashboard splat route
	}
}

// setupRouter creates a router with standard test routes
func setupRouter(scale string) *Router {
	router := &Router{}

	switch scale {
	case "small":
		// Basic routes for simple tests
		router.AddRoute("/")
		router.AddRoute("/users")
		router.AddRoute("/users/$id")
		router.AddRoute("/users/$id/profile")
		router.AddRoute("/api/v1/users")
		router.AddRoute("/api/$version/users")
		router.AddRoute("/api/v1/users/$id")
		router.AddRoute("/files/$")

	case "medium":
		// RESTful API-style routes
		for i := 0; i < 1000; i++ {
			router.AddRoute(fmt.Sprintf("/api/v%d/users", i%5))
			router.AddRoute(fmt.Sprintf("/api/v%d/users/$id", i%5))
			router.AddRoute(fmt.Sprintf("/api/v%d/users/$id/posts/$post_id", i%5))
			router.AddRoute(fmt.Sprintf("/files/bucket%d/$", i%10))
		}

	case "large":
		// Complex application routes
		for i := 0; i < 10000; i++ {
			// Static routes
			router.AddRoute(fmt.Sprintf("/api/v%d/users", i%10))
			router.AddRoute(fmt.Sprintf("/api/v%d/products", i%10))
			router.AddRoute(fmt.Sprintf("/docs/section%d", i%100))

			// Dynamic routes
			router.AddRoute(fmt.Sprintf("/api/v%d/users/$id/posts/$post_id", i%10))
			router.AddRoute(fmt.Sprintf("/api/v%d/products/$category/$id", i%10))

			// Splat routes
			router.AddRoute(fmt.Sprintf("/files/bucket%d/$", i%20))
		}
	}

	return router
}

// generateTestPaths creates test paths for different scenarios
func generateTestPaths(scale string) []string {
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
		for i := 0; i < 400; i++ {
			paths = append(paths, fmt.Sprintf("/api/v%d/users", i%5))
		}

		// Dynamic paths (40%)
		for i := 0; i < 400; i++ {
			paths = append(paths, fmt.Sprintf("/api/v%d/users/%d/posts/%d", i%5, i, i%100))
		}

		// Splat paths (20%)
		for i := 0; i < 200; i++ {
			paths = append(paths, fmt.Sprintf("/files/bucket%d/path/to/file%d.txt", i%10, i))
		}

		return paths
	}
	return nil
}

// Compare nested router performance
func BenchmarkFindAllMatches(b *testing.B) {
	cases := []struct {
		name     string
		pathType string
		paths    []string
	}{
		{
			name:     "StaticRoutes",
			pathType: "static",
			paths:    []string{"/", "/dashboard", "/dashboard/customers", "/tiger", "/lion"},
		},
		{
			name:     "DynamicRoutes",
			pathType: "dynamic",
			paths: []string{
				"/dashboard/customers/123",
				"/dashboard/customers/456/orders",
				"/tiger/123",
				"/bear/123",
			},
		},
		{
			name:     "DeepNestedRoutes",
			pathType: "deep",
			paths: []string{
				"/dashboard/customers/123/orders/456",
				"/tiger/123/456/789",
				"/bear/123/456/789",
				"/articles/test/articles",
			},
		},
		{
			name:     "SplatRoutes",
			pathType: "splat",
			paths: []string{
				"/does-not-exist",
				"/dashboard/unknown/path",
				"/tiger/123/456/789/extra",
				"/bear/123/456/789/extra",
			},
		},
		{
			name:     "MixedRoutes",
			pathType: "mixed",
			paths:    generateNestedTestPaths(),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			router := setupNestedRouter()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				path := tc.paths[i%len(tc.paths)]
				matches, _ := router.FindAllMatches(path)
				runtime.KeepAlive(matches)
			}
		})
	}
}

// Compare scaling performance
func BenchmarkFindBestMatchAtScale(b *testing.B) {
	scales := []string{"small", "medium", "large"}

	for _, scale := range scales {
		b.Run(fmt.Sprintf("Scale_%s", scale), func(b *testing.B) {
			router := setupRouter(scale)
			paths := generateTestPaths(scale)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				path := paths[i%len(paths)]
				match, _ := router.FindBestMatch(path)
				runtime.KeepAlive(match)
			}
		})
	}

	b.Run("WorstCase_DeepNested", func(b *testing.B) {
		router := setupRouter("large")
		path := "/api/v9/users/999/posts/999"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			match, _ := router.FindBestMatch(path)
			runtime.KeepAlive(match)
		}
	})
}

// Compare router metrics
func BenchmarkFindBestMatchWithMetrics(b *testing.B) {
	scenarios := []struct {
		name string
		path string
	}{
		{"StaticRoute", "/api/v1/users"},
		{"DynamicRoute", "/api/v1/users/123/posts/456"},
		{"SplatRoute", "/files/bucket1/deep/path/file.txt"},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			var memStatsBefore runtime.MemStats
			runtime.ReadMemStats(&memStatsBefore)

			router := setupRouter("medium")

			runtime.GC()
			var memStatsAfter runtime.MemStats
			runtime.ReadMemStats(&memStatsAfter)
			routerMemory := memStatsAfter.HeapAlloc - memStatsBefore.HeapAlloc

			b.ResetTimer()
			b.ReportAllocs()

			matches := 0
			for i := 0; i < b.N; i++ {
				match, ok := router.FindBestMatch(s.path)
				if ok {
					matches++
				}
				runtime.KeepAlive(match)
			}

			b.ReportMetric(float64(routerMemory), "router_bytes")
			b.ReportMetric(float64(matches)/float64(b.N), "match_ratio")
		})
	}
}

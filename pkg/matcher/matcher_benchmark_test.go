package matcher

import (
	"testing"

	"github.com/sjc5/kit/pkg/router"
)

// setupBenchmarkPaths creates a realistic set of paths for benchmarking
func setupBenchmarkPaths() []*RegisteredPath {
	patterns := []string{
		"/$",
		"/_index",
		"/api/v1/users/_index",
		"/api/v1/users/$id",
		"/api/v1/users/$id/profile",
		"/api/v1/users/$id/posts/_index",
		"/api/v1/users/$id/posts/$post_id",
		"/api/v1/users/$id/posts/$post_id/comments/_index",
		"/api/v1/users/$id/posts/$post_id/comments/$comment_id",
		"/api/v1/posts/_index",
		"/api/v1/posts/$id",
		"/api/v1/posts/$id/comments/$",
		"/docs/$",
		"/blog/_index",
		"/blog/$slug",
		"/blog/categories/$category/_index",
		"/products/_index",
		"/products/$category/_index",
		"/products/$category/$product_id",
		"/dashboard",
		"/dashboard/$",
		"/dashboard/analytics",
		"/dashboard/settings",
		"/dashboard/users/$id",
		"/static/$",
	}

	var rps []*RegisteredPath

	for _, pattern := range patterns {
		rps = append(rps, PatternToRegisteredPath(pattern))
	}

	return rps
}

// BenchmarkGetMatchingPaths benchmarks the performance of the path matching
func BenchmarkGetMatchingPaths(b *testing.B) {
	paths := setupBenchmarkPaths()
	benchCases := []struct {
		name string
		path string
	}{
		{"RootPath", "/"},
		{"StaticPath", "/dashboard"},
		{"DynamicPath", "/api/v1/users/123"},
		{"DeepPath", "/api/v1/users/123/posts/456/comments/789"},
		{"SplatPath", "/docs/getting-started/introduction"},
		{"NonExistentPath", "/this/does/not/exist"},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				GetMatchingPaths(paths, bc.path)
			}
		})
	}
}

// BenchmarkMatcherCore benchmarks the core matching function
func BenchmarkMatcherCore(b *testing.B) {
	benchCases := []struct {
		name     string
		pattern  string
		realPath string
	}{
		{"ExactMatch", "/test", "/test"},
		{"DynamicMatch", "/users/$id", "/users/123"},
		{"SplatMatch", "/files/$", "/files/documents/report.pdf"},
		{"ComplexMatch", "/api/v1/users/$id/posts/$post_id", "/api/v1/users/123/posts/456"},
	}

	for _, bc := range benchCases {
		rp := PatternToRegisteredPath(bc.pattern)
		realSegments := router.ParseSegments(bc.realPath)

		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				matchCore(rp.Segments, realSegments)
			}
		})
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
		_, _ = matchCore(patterns[patternIdx], paths[pathIdx])
	}
}

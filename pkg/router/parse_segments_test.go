package router

import (
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
		{"empty path", "", []string{}},
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

func BenchmarkParseSegments(b *testing.B) {
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

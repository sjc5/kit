package router

import (
	"reflect"
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

package router

import (
	"fmt"
	"testing"
)

func TestWildcardMatching(t *testing.T) {
	tests := []struct {
		pattern               string
		path                  string
		shouldMatch           bool
		expectedParams        Params
		expectedSplatSegments []string
	}{
		{"/bob/hi/there", "/bob/hi/there/dude", true, nil, []string{}},
		{"/bob/$", "/bob/hi/there/dude", true, nil, []string{"hi", "there", "dude"}},
		{"/bob/$", "/bob/hi", true, Params{}, []string{"hi"}},
		{"/users/$userId", "/users/123", true, Params{"userId": "123"}, []string{}},
		{"/api/$version/users/$id", "/api/v1/users/42", true, Params{"version": "v1", "id": "42"}, []string{}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s   matching   %s", test.pattern, test.path), func(t *testing.T) {
			patternSegments := ParseSegments(test.pattern)
			pathSegments := ParseSegments(test.path)

			result, matched := MatchCore(patternSegments, pathSegments)

			if matched != test.shouldMatch {
				t.Errorf("Expected match: %v, got: %v", test.shouldMatch, matched)
				return
			}

			if !matched {
				return
			}

			if len(result.Params) != len(test.expectedParams) {
				t.Errorf("Expected params count: %d, got: %d", len(test.expectedParams), len(result.Params))
				return
			}

			for k, v := range test.expectedParams {
				if result.Params[k] != v {
					t.Errorf("Expected param %s: %s, got: %s", k, v, result.Params[k])
				}
			}

			if len(result.SplatSegments) != len(test.expectedSplatSegments) {
				t.Errorf("Expected splat segments count: %d, got: %d", len(test.expectedSplatSegments), len(result.SplatSegments))
				return
			}

			for i, v := range test.expectedSplatSegments {
				if result.SplatSegments[i] != v {
					t.Errorf("Expected splat segment %d: %s, got: %s", i, v, result.SplatSegments[i])
				}
			}
		})
	}
}

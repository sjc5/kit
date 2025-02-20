package matcher

import (
	"reflect"
	"testing"

	"github.com/sjc5/kit/pkg/router"
)

// rawPatterns_PreBuild contains all the route patterns to test -- have not been run through PatternToRegisteredPath
var rawPatterns_PreBuild = []string{
	"/$",
	"/_index",
	"/articles/_index",
	"/articles/test/articles/_index",
	"/bear/$bear_id/$",
	"/bear/$bear_id",
	"/bear/_index",
	"/bear",
	"/dashboard/$",
	"/dashboard/_index",
	"/dashboard/customers/$customer_id/_index",
	"/dashboard/customers/$customer_id/orders/$order_id",
	"/dashboard/customers/$customer_id/orders/_index",
	"/dashboard/customers/$customer_id/orders",
	"/dashboard/customers/$customer_id",
	"/dashboard/customers/_index",
	"/dashboard/customers",
	"/dashboard",
	"/dynamic-index/$pagename/_index",
	"/dynamic-index/__site_index/index",
	"/lion/$",
	"/lion/_index",
	"/lion",
	"/tiger/$tiger_id/$",
	"/tiger/$tiger_id/$tiger_cub_id",
	"/tiger/$tiger_id/_index",
	"/tiger/$tiger_id",
	"/tiger/_index",
	"/tiger",
}

// expectedRegisteredPaths_PostBuild contains the expected RegisteredPath objects for each pattern
var expectedRegisteredPaths_PostBuild = []*RegisteredPath{
	{Pattern: "/$", Segments: []string{"$"}, PathType: PathTypes.UltimateCatch},
	{Pattern: "/_index", Segments: []string{""}, PathType: PathTypes.Index},
	{Pattern: "/articles/_index", Segments: []string{"articles", ""}, PathType: PathTypes.Index},
	{Pattern: "/articles/test/articles/_index", Segments: []string{"articles", "test", "articles", ""}, PathType: PathTypes.Index},
	{Pattern: "/bear/$bear_id/$", Segments: []string{"bear", "$bear_id", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/bear/$bear_id", Segments: []string{"bear", "$bear_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/bear/_index", Segments: []string{"bear", ""}, PathType: PathTypes.Index},
	{Pattern: "/bear", Segments: []string{"bear"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/$", Segments: []string{"dashboard", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/dashboard/_index", Segments: []string{"dashboard", ""}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/_index", Segments: []string{"dashboard", "customers", "$customer_id", ""}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/orders/$order_id", Segments: []string{"dashboard", "customers", "$customer_id", "orders", "$order_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/dashboard/customers/$customer_id/orders/_index", Segments: []string{"dashboard", "customers", "$customer_id", "orders", ""}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/orders", Segments: []string{"dashboard", "customers", "$customer_id", "orders"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/customers/$customer_id", Segments: []string{"dashboard", "customers", "$customer_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/dashboard/customers/_index", Segments: []string{"dashboard", "customers", ""}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers", Segments: []string{"dashboard", "customers"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard", Segments: []string{"dashboard"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dynamic-index/$pagename/_index", Segments: []string{"dynamic-index", "$pagename", ""}, PathType: PathTypes.Index},
	{Pattern: "/dynamic-index/index", Segments: []string{"dynamic-index", "index"}, PathType: PathTypes.StaticLayout}, // PatternToRegisteredPath strips out segments starting with double underscores
	{Pattern: "/lion/$", Segments: []string{"lion", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/lion/_index", Segments: []string{"lion", ""}, PathType: PathTypes.Index},
	{Pattern: "/lion", Segments: []string{"lion"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/tiger/$tiger_id/$", Segments: []string{"tiger", "$tiger_id", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/tiger/$tiger_id/$tiger_cub_id", Segments: []string{"tiger", "$tiger_id", "$tiger_cub_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/$tiger_id/_index", Segments: []string{"tiger", "$tiger_id", ""}, PathType: PathTypes.Index},
	{Pattern: "/tiger/$tiger_id", Segments: []string{"tiger", "$tiger_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/_index", Segments: []string{"tiger", ""}, PathType: PathTypes.Index},
	{Pattern: "/tiger", Segments: []string{"tiger"}, PathType: PathTypes.StaticLayout},
}

type ExpectedMatches map[string]struct {
	ExpectedScore     int
	ExpectedSegLength int
}

// TestPathScenarios defines test scenarios for GetMatchingPaths
type TestPathScenario struct {
	Path              string
	ExpectedPathTypes []PathType
	ExpectedParams    Params
	ExpectedSplatSegs []string
	ExpectedMatches   ExpectedMatches
}

// PathScenarios contains all the test scenarios
var PathScenarios = []TestPathScenario{
	{
		Path:              "/does-not-exist",
		ExpectedPathTypes: []PathType{PathTypes.UltimateCatch},
		ExpectedSplatSegs: []string{"does-not-exist"},
		ExpectedMatches: ExpectedMatches{
			"/$": {ExpectedScore: 1, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/this-should-be-ignored",
		ExpectedPathTypes: []PathType{PathTypes.UltimateCatch},
		ExpectedSplatSegs: []string{"this-should-be-ignored"},
		ExpectedMatches: ExpectedMatches{
			"/$": {ExpectedScore: 1, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/_index": {ExpectedScore: 0, ExpectedSegLength: 0},
		},
	},
	{
		Path:              "/lion",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/lion":        {ExpectedScore: 3, ExpectedSegLength: 1},
			"/lion/_index": {ExpectedScore: 3, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/lion/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.NonUltimateSplat},
		ExpectedSplatSegs: []string{"123"},
		ExpectedMatches: ExpectedMatches{
			"/lion":   {ExpectedScore: 3, ExpectedSegLength: 2},
			"/lion/$": {ExpectedScore: 4, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/lion/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.NonUltimateSplat},
		ExpectedSplatSegs: []string{"123", "456"},
		ExpectedMatches: ExpectedMatches{
			"/lion":   {ExpectedScore: 3, ExpectedSegLength: 3},
			"/lion/$": {ExpectedScore: 4, ExpectedSegLength: 3},
		},
	},
	{
		Path:              "/lion/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.NonUltimateSplat},
		ExpectedSplatSegs: []string{"123", "456", "789"},
		ExpectedMatches: ExpectedMatches{
			"/lion":   {ExpectedScore: 3, ExpectedSegLength: 4},
			"/lion/$": {ExpectedScore: 4, ExpectedSegLength: 4},
		},
	},
	{
		Path:              "/tiger",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/tiger":        {ExpectedScore: 3, ExpectedSegLength: 1},
			"/tiger/_index": {ExpectedScore: 3, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/tiger/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Index},
		ExpectedParams:    Params{"tiger_id": "123"},
		ExpectedMatches: ExpectedMatches{
			"/tiger":                  {ExpectedScore: 3, ExpectedSegLength: 2},
			"/tiger/$tiger_id":        {ExpectedScore: 5, ExpectedSegLength: 2},
			"/tiger/$tiger_id/_index": {ExpectedScore: 5, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/tiger/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"tiger_id": "123", "tiger_cub_id": "456"},
		ExpectedMatches: ExpectedMatches{
			"/tiger":                         {ExpectedScore: 3, ExpectedSegLength: 3},
			"/tiger/$tiger_id":               {ExpectedScore: 5, ExpectedSegLength: 3},
			"/tiger/$tiger_id/$tiger_cub_id": {ExpectedScore: 7, ExpectedSegLength: 3},
		},
	},
	{
		Path:              "/tiger/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.NonUltimateSplat},
		ExpectedParams:    Params{"tiger_id": "123"},
		ExpectedSplatSegs: []string{"456", "789"},
		ExpectedMatches: ExpectedMatches{
			"/tiger":             {ExpectedScore: 3, ExpectedSegLength: 4},
			"/tiger/$tiger_id":   {ExpectedScore: 5, ExpectedSegLength: 4},
			"/tiger/$tiger_id/$": {ExpectedScore: 6, ExpectedSegLength: 4},
		},
	},
	{
		Path:              "/bear",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/bear":        {ExpectedScore: 3, ExpectedSegLength: 1},
			"/bear/_index": {ExpectedScore: 3, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/bear/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedMatches: ExpectedMatches{
			"/bear":          {ExpectedScore: 3, ExpectedSegLength: 2},
			"/bear/$bear_id": {ExpectedScore: 5, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/bear/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.NonUltimateSplat},
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedSplatSegs: []string{"456"},
		ExpectedMatches: ExpectedMatches{
			"/bear":            {ExpectedScore: 3, ExpectedSegLength: 3},
			"/bear/$bear_id":   {ExpectedScore: 5, ExpectedSegLength: 3},
			"/bear/$bear_id/$": {ExpectedScore: 6, ExpectedSegLength: 3},
		},
	},
	{
		Path:              "/bear/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.NonUltimateSplat},
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedSplatSegs: []string{"456", "789"},
		ExpectedMatches: ExpectedMatches{
			"/bear":            {ExpectedScore: 3, ExpectedSegLength: 4},
			"/bear/$bear_id":   {ExpectedScore: 5, ExpectedSegLength: 4},
			"/bear/$bear_id/$": {ExpectedScore: 6, ExpectedSegLength: 4},
		},
	},
	{
		Path:              "/dashboard",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":        {ExpectedScore: 3, ExpectedSegLength: 1},
			"/dashboard/_index": {ExpectedScore: 3, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/dashboard/asdf",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.NonUltimateSplat},
		ExpectedSplatSegs: []string{"asdf"},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":   {ExpectedScore: 3, ExpectedSegLength: 2},
			"/dashboard/$": {ExpectedScore: 4, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/dashboard/customers",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":                  {ExpectedScore: 3, ExpectedSegLength: 2},
			"/dashboard/customers":        {ExpectedScore: 6, ExpectedSegLength: 2},
			"/dashboard/customers/_index": {ExpectedScore: 6, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/dashboard/customers/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Index},
		ExpectedParams:    Params{"customer_id": "123"},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":                               {ExpectedScore: 3, ExpectedSegLength: 3},
			"/dashboard/customers":                     {ExpectedScore: 6, ExpectedSegLength: 3},
			"/dashboard/customers/$customer_id":        {ExpectedScore: 8, ExpectedSegLength: 3},
			"/dashboard/customers/$customer_id/_index": {ExpectedScore: 8, ExpectedSegLength: 3},
		},
	},
	{
		Path:              "/dashboard/customers/123/orders",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.StaticLayout, PathTypes.Index},
		ExpectedParams:    Params{"customer_id": "123"},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":                                      {ExpectedScore: 3, ExpectedSegLength: 4},
			"/dashboard/customers":                            {ExpectedScore: 6, ExpectedSegLength: 4},
			"/dashboard/customers/$customer_id":               {ExpectedScore: 8, ExpectedSegLength: 4},
			"/dashboard/customers/$customer_id/orders":        {ExpectedScore: 11, ExpectedSegLength: 4},
			"/dashboard/customers/$customer_id/orders/_index": {ExpectedScore: 11, ExpectedSegLength: 4},
		},
	},
	{
		Path:              "/dashboard/customers/123/orders/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"customer_id": "123", "order_id": "456"},
		ExpectedMatches: ExpectedMatches{
			"/dashboard":                                         {ExpectedScore: 3, ExpectedSegLength: 5},
			"/dashboard/customers":                               {ExpectedScore: 6, ExpectedSegLength: 5},
			"/dashboard/customers/$customer_id":                  {ExpectedScore: 8, ExpectedSegLength: 5},
			"/dashboard/customers/$customer_id/orders":           {ExpectedScore: 11, ExpectedSegLength: 5},
			"/dashboard/customers/$customer_id/orders/$order_id": {ExpectedScore: 13, ExpectedSegLength: 5},
		},
	},
	{
		Path:              "/articles",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/articles/_index": {ExpectedScore: 3, ExpectedSegLength: 1},
		},
	},
	{
		Path:              "/articles/bob",
		ExpectedPathTypes: []PathType{PathTypes.UltimateCatch},
		ExpectedSplatSegs: []string{"articles", "bob"},
		ExpectedMatches: ExpectedMatches{
			"/$": {ExpectedScore: 1, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/articles/test",
		ExpectedPathTypes: []PathType{PathTypes.UltimateCatch},
		ExpectedSplatSegs: []string{"articles", "test"},
		ExpectedMatches: ExpectedMatches{
			"/$": {ExpectedScore: 1, ExpectedSegLength: 2},
		},
	},
	{
		Path:              "/articles/test/articles",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			"/articles/test/articles/_index": {ExpectedScore: 9, ExpectedSegLength: 3},
		},
	},
	{
		Path:              "/dynamic-index/index",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout},
		ExpectedMatches: ExpectedMatches{
			"/dynamic-index/index": {ExpectedScore: 6, ExpectedSegLength: 2},
		},
	},
}

// the RegisteredPaths fed into GetMatchingPaths should always have been generated using PatternToRegisteredPath
func TestGetMatchingPaths(t *testing.T) {
	for _, tc := range PathScenarios {
		t.Run(tc.Path, func(t *testing.T) {
			splatSegs, matches := GetMatchingPaths(expectedRegisteredPaths_PostBuild, tc.Path)

			// Verify correct number of matching paths
			if len(matches) != len(tc.ExpectedPathTypes) {
				t.Errorf("Expected %d matching paths, got %d", len(tc.ExpectedPathTypes), len(matches))
				for i, m := range matches {
					t.Logf("Match %d: %s (Score: %d)", i, m.Pattern, m.Results.Score)
				}
				return // Fail fast if count doesn't match
			}

			// Track scores to detect duplicates
			scoreCounts := make(map[int]int)

			// Verify each matching path's properties in order
			for i, match := range matches {
				// Track occurrences of scores
				scoreCounts[match.Results.Score]++

				// Verify PathType (in correct order)
				expectedPathType := tc.ExpectedPathTypes[i]
				if match.PathType != expectedPathType {
					t.Errorf("Match %d: expected PathType %s, got %s (matches returned in wrong order)",
						i, expectedPathType, match.PathType)
				}

				// Verify Pattern exists in ExpectedMatches
				expectedData, exists := tc.ExpectedMatches[match.Pattern]
				if !exists {
					t.Errorf("Match %d: pattern %s not found in ExpectedMatches",
						i, match.Pattern)
					continue
				}

				// Verify Results exists
				if match.Results == nil {
					t.Errorf("Match %d: Results is nil", i)
					continue
				}

				// Verify Score
				if match.Results.Score != expectedData.ExpectedScore {
					t.Errorf("Match %d: expected Score %d, got %d",
						i, expectedData.ExpectedScore, match.Results.Score)
				}

				// Verify RealSegmentsLength
				if match.Results.RealSegmentsLength != expectedData.ExpectedSegLength {
					t.Errorf("Match %d: expected RealSegmentsLength %d, got %d",
						i, expectedData.ExpectedSegLength, match.Results.RealSegmentsLength)
				}

				// Find matching registered path for segment verification
				found := false
				for _, rp := range expectedRegisteredPaths_PostBuild {
					if rp.Pattern == match.Pattern {
						// Verify Segments
						if !reflect.DeepEqual(match.Segments, rp.Segments) {
							t.Errorf("Match %d: expected Segments %v, got %v",
								i, rp.Segments, match.Segments)
						}
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Match %d: couldn't find registered path with pattern %s",
						i, match.Pattern)
				}
			}

			// Verify paths are in correct order (sorted by segment length)
			for i := 1; i < len(matches); i++ {
				if len(matches[i-1].Segments) > len(matches[i].Segments) {
					t.Errorf("Matches not properly sorted: match[%d] has %d segments but match[%d] has %d segments",
						i-1, len(matches[i-1].Segments), i, len(matches[i].Segments))
				}
			}

			// Verify params
			if tc.ExpectedParams != nil {
				combinedParams := make(Params)
				for _, match := range matches {
					if match.Results != nil && match.Results.Params != nil {
						for k, v := range match.Results.Params {
							combinedParams[k] = v
						}
					}
				}

				if !reflect.DeepEqual(combinedParams, tc.ExpectedParams) {
					t.Errorf("Expected params %v, got %v", tc.ExpectedParams, combinedParams)
				}
			} else if len(matches) > 0 && hasAnyParams(matches) {
				t.Errorf("Found params in matches but none were expected")
			}

			// Verify splat segments
			if tc.ExpectedSplatSegs != nil {
				if !compareStringSlices(splatSegs, tc.ExpectedSplatSegs) {
					t.Errorf("Expected splat segments %v, got %v", tc.ExpectedSplatSegs, splatSegs)
				}
			} else if len(splatSegs) > 0 {
				t.Errorf("Found splat segments %v but none were expected", splatSegs)
			}

			// Check for duplicate scores
			for score, count := range scoreCounts {
				if count > 1 {
					t.Logf("Warning: Multiple matches with the same score (%d) for path %q", score, tc.Path)
				}
			}
		})
	}
}

// Helper function to check if any match has params
func hasAnyParams(matches []*Match) bool {
	for _, match := range matches {
		if match.Results != nil && match.Results.Params != nil && len(match.Results.Params) > 0 {
			return true
		}
	}
	return false
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func TestMatcherCore(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		realPath       string
		expectedResult *Results
		expectedMatch  bool
	}{
		{
			name:     "exact match",
			pattern:  "/test",
			realPath: "/test",
			expectedResult: &Results{
				Params:             Params{},
				Score:              3,
				RealSegmentsLength: 1,
			},
			expectedMatch: true,
		},
		{
			name:     "dynamic parameter match",
			pattern:  "/users/$id",
			realPath: "/users/123",
			expectedResult: &Results{
				Params:             Params{"id": "123"},
				Score:              5, // 3 for "users" + 2 for "$id"
				RealSegmentsLength: 2,
			},
			expectedMatch: true,
		},
		{
			name:     "catch-all splat match",
			pattern:  "/files/$",
			realPath: "/files/documents/report.pdf",
			expectedResult: &Results{
				Params:             Params{},
				Score:              4, // 3 for "files" + 1 for "$"
				RealSegmentsLength: 3,
			},
			expectedMatch: true,
		},
		{
			name:           "no match - different segments",
			pattern:        "/users",
			realPath:       "/posts",
			expectedResult: nil,
			expectedMatch:  false,
		},
		{
			name:           "no match - pattern longer than path",
			pattern:        "/users/$id/profile",
			realPath:       "/users/123",
			expectedResult: nil,
			expectedMatch:  false,
		},
		{
			name:     "index route match",
			pattern:  "/_index",
			realPath: "/",
			expectedResult: &Results{
				Params:             Params{},
				Score:              0,
				RealSegmentsLength: 0,
			},
			expectedMatch: true,
		},
		{
			name:     "complex nested path with parameters",
			pattern:  "/dashboard/customers/$customer_id/orders/$order_id",
			realPath: "/dashboard/customers/abc123/orders/xyz789",
			expectedResult: &Results{
				Params: Params{
					"customer_id": "abc123",
					"order_id":    "xyz789",
				},
				Score:              13, // 3*3 for static segments + 2*2 for dynamic segments
				RealSegmentsLength: 5,
			},
			expectedMatch: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rp := PatternToRegisteredPath(tc.pattern)
			realSegments := router.ParseSegments(tc.realPath)

			result, ok := MatchCoreWithPrep(rp.Segments, realSegments)

			if ok != tc.expectedMatch {
				t.Errorf("Expected match: %v, got: %v", tc.expectedMatch, ok)
			}

			if !tc.expectedMatch {
				if result != nil {
					t.Errorf("Expected nil result for non-match, got: %v", result)
				}
				return
			}

			// Verify params
			if !reflect.DeepEqual(result.Params, tc.expectedResult.Params) {
				t.Errorf("Expected params %v, got %v", tc.expectedResult.Params, result.Params)
			}

			// Verify score
			if result.Score != tc.expectedResult.Score {
				t.Errorf("Expected score %d, got %d", tc.expectedResult.Score, result.Score)
			}

			// Verify real segments length
			if result.RealSegmentsLength != tc.expectedResult.RealSegmentsLength {
				t.Errorf("Expected real segments length %d, got %d",
					tc.expectedResult.RealSegmentsLength, result.RealSegmentsLength)
			}
		})
	}
}

func TestPatternToRegisteredPath(t *testing.T) {
	for i, pattern := range rawPatterns_PreBuild {
		t.Run(pattern, func(t *testing.T) {
			expected := expectedRegisteredPaths_PostBuild[i]
			result := PatternToRegisteredPath(pattern)

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("PatternToRegisteredPath(%q)\n got: %+v\nwant: %+v", pattern, result, expected)

				if result.Pattern != expected.Pattern {
					t.Errorf("  Pattern: got %q, want %q", result.Pattern, expected.Pattern)
				}
				if !reflect.DeepEqual(result.Segments, expected.Segments) {
					t.Errorf("  Segments: got %v, want %v", result.Segments, expected.Segments)
				}
				if result.PathType != expected.PathType {
					t.Errorf("  PathType: got %v, want %v", result.PathType, expected.PathType)
				}
			}
		})
	}
}

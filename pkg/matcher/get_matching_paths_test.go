package matcher

import (
	"reflect"
	"testing"
)

var finalRegisteredPathsForTest = []*RegisteredPath{
	{Pattern: "/", PathType: PathTypes.Index},
	{Pattern: "/articles", PathType: PathTypes.Index},
	{Pattern: "/articles/test/articles", PathType: PathTypes.Index},
	{Pattern: "/bear", PathType: PathTypes.Index},
	{Pattern: "/dashboard", PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers", PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id", PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/orders", PathType: PathTypes.Index},
	{Pattern: "/dynamic-index/$pagename", PathType: PathTypes.Index},
	{Pattern: "/lion", PathType: PathTypes.Index},
	{Pattern: "/tiger", PathType: PathTypes.Index},
	{Pattern: "/tiger/$tiger_id", PathType: PathTypes.Index},

	{Pattern: "/$", PathType: PathTypes.Splat}, // ultimate catch
	{Pattern: "/bear", PathType: PathTypes.StaticLayout},
	{Pattern: "/bear/$bear_id", PathType: PathTypes.DynamicLayout},
	{Pattern: "/bear/$bear_id/$", PathType: PathTypes.Splat}, // non-ultimate splat
	{Pattern: "/dashboard", PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/$", PathType: PathTypes.Splat}, // non-ultimate splat
	{Pattern: "/dashboard/customers", PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/customers/$customer_id", PathType: PathTypes.DynamicLayout},
	{Pattern: "/dashboard/customers/$customer_id/orders", PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/customers/$customer_id/orders/$order_id", PathType: PathTypes.DynamicLayout},
	// PatternToRegisteredPath strips out segments starting with double underscores
	{Pattern: "/dynamic-index/index", PathType: PathTypes.StaticLayout},
	{Pattern: "/lion", PathType: PathTypes.StaticLayout},
	{Pattern: "/lion/$", PathType: PathTypes.Splat}, // non-ultimate splat
	{Pattern: "/tiger", PathType: PathTypes.StaticLayout},
	{Pattern: "/tiger/$tiger_id", PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/$tiger_id/$tiger_cub_id", PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/$tiger_id/$", PathType: PathTypes.Splat}, // non-ultimate splat
}

// TestPathScenarios defines test scenarios for GetMatchingPaths
type TestPathScenario struct {
	Path              string
	ExpectedPathTypes []PathType
	ExpectedParams    Params
	ExpectedSplatSegs []string
	ExpectedMatches   ExpectedMatches
}

type expectedMatch struct {
	Pattern string
}

type ExpectedMatches []expectedMatch

// PathScenarios contains all the test scenarios
var PathScenarios = []TestPathScenario{
	{
		Path:              "/does-not-exist",
		ExpectedPathTypes: []PathType{PathTypes.Splat}, // ultimate catch
		ExpectedSplatSegs: []string{"does-not-exist"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/$"},
		},
	},
	{
		Path:              "/this-should-be-ignored",
		ExpectedPathTypes: []PathType{PathTypes.Splat}, // ultimate catch
		ExpectedSplatSegs: []string{"this-should-be-ignored"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/$"},
		},
	},
	{
		Path:              "/",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/"},
		},
	},
	{
		Path:              "/lion",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/lion"},
			{Pattern: "/lion"},
		},
	},
	{
		Path:              "/lion/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedSplatSegs: []string{"123"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/lion"},
			{Pattern: "/lion/$"},
		},
	},
	{
		Path:              "/lion/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedSplatSegs: []string{"123", "456"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/lion"},
			{Pattern: "/lion/$"},
		},
	},
	{
		Path:              "/lion/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedSplatSegs: []string{"123", "456", "789"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/lion"},
			{Pattern: "/lion/$"},
		},
	},
	{
		Path:              "/tiger",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/tiger"},
			{Pattern: "/tiger"},
		},
	},
	{
		Path:              "/tiger/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Index},
		ExpectedParams:    Params{"tiger_id": "123"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/tiger"},
			{Pattern: "/tiger/$tiger_id"},
			{Pattern: "/tiger/$tiger_id"},
		},
	},
	{
		Path:              "/tiger/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"tiger_id": "123", "tiger_cub_id": "456"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/tiger"},
			{Pattern: "/tiger/$tiger_id"},
			{Pattern: "/tiger/$tiger_id/$tiger_cub_id"},
		},
	},
	{
		Path:              "/tiger/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedParams:    Params{"tiger_id": "123"},
		ExpectedSplatSegs: []string{"456", "789"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/tiger"},
			{Pattern: "/tiger/$tiger_id"},
			{Pattern: "/tiger/$tiger_id/$"},
		},
	},
	{
		Path:              "/bear",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/bear"},
			{Pattern: "/bear"},
		},
	},
	{
		Path:              "/bear/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/bear"},
			{Pattern: "/bear/$bear_id"},
		},
	},
	{
		Path:              "/bear/123/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedSplatSegs: []string{"456"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/bear"},
			{Pattern: "/bear/$bear_id"},
			{Pattern: "/bear/$bear_id/$"},
		},
	},
	{
		Path:              "/bear/123/456/789",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedParams:    Params{"bear_id": "123"},
		ExpectedSplatSegs: []string{"456", "789"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/bear"},
			{Pattern: "/bear/$bear_id"},
			{Pattern: "/bear/$bear_id/$"},
		},
	},
	{
		Path:              "/dashboard",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard"},
		},
	},
	{
		Path:              "/dashboard/asdf",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.Splat}, // non-ultimate splat
		ExpectedSplatSegs: []string{"asdf"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard/$"},
		},
	},
	{
		Path:              "/dashboard/customers",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard/customers"},
			{Pattern: "/dashboard/customers"},
		},
	},
	{
		Path:              "/dashboard/customers/123",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.Index},
		ExpectedParams:    Params{"customer_id": "123"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard/customers"},
			{Pattern: "/dashboard/customers/$customer_id"},
			{Pattern: "/dashboard/customers/$customer_id"},
		},
	},
	{
		Path:              "/dashboard/customers/123/orders",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.StaticLayout, PathTypes.Index},
		ExpectedParams:    Params{"customer_id": "123"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard/customers"},
			{Pattern: "/dashboard/customers/$customer_id"},
			{Pattern: "/dashboard/customers/$customer_id/orders"},
			{Pattern: "/dashboard/customers/$customer_id/orders"},
		},
	},
	{
		Path:              "/dashboard/customers/123/orders/456",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout, PathTypes.StaticLayout, PathTypes.DynamicLayout},
		ExpectedParams:    Params{"customer_id": "123", "order_id": "456"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dashboard"},
			{Pattern: "/dashboard/customers"},
			{Pattern: "/dashboard/customers/$customer_id"},
			{Pattern: "/dashboard/customers/$customer_id/orders"},
			{Pattern: "/dashboard/customers/$customer_id/orders/$order_id"},
		},
	},
	{
		Path:              "/articles",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/articles"},
		},
	},
	{
		Path:              "/articles/bob",
		ExpectedPathTypes: []PathType{PathTypes.Splat}, // ultimate catch
		ExpectedSplatSegs: []string{"articles", "bob"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/$"},
		},
	},
	{
		Path:              "/articles/test",
		ExpectedPathTypes: []PathType{PathTypes.Splat}, // ultimate catch
		ExpectedSplatSegs: []string{"articles", "test"},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/$"},
		},
	},
	{
		Path:              "/articles/test/articles",
		ExpectedPathTypes: []PathType{PathTypes.Index},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/articles/test/articles"},
		},
	},
	{
		Path:              "/dynamic-index/index",
		ExpectedPathTypes: []PathType{PathTypes.StaticLayout},
		ExpectedMatches: ExpectedMatches{
			{Pattern: "/dynamic-index/index"},
		},
	},
}

// the RegisteredPaths fed into GetMatchingPaths should always have been generated using PatternToRegisteredPath
func TestGetMatchingPaths(t *testing.T) {
	for _, tc := range PathScenarios {
		t.Run(tc.Path, func(t *testing.T) {
			splatSegs, matches := GetMatchingPaths(finalRegisteredPathsForTest, tc.Path)

			// Verify correct number of matching paths
			if len(matches) != len(tc.ExpectedPathTypes) {
				t.Errorf("Expected %d matching paths, got %d", len(tc.ExpectedPathTypes), len(matches))

				// print expected matches
				for i, pattern := range tc.ExpectedMatches {
					t.Logf("Expected match %d: %s PathType: %s", i+1, pattern.Pattern, tc.ExpectedPathTypes[i])
					i++
				}

				for i, m := range matches {
					t.Logf("Match %d: %s (Score: %d) PathType: %s", i+1, m.Pattern, m.Results.Score, m.PathType)
				}
				return // Fail fast if count doesn't match
			}

			// Verify each matching path's properties in order
			for i, match := range matches {
				// Verify PathType (in correct order)
				expectedPathType := tc.ExpectedPathTypes[i]
				if match.PathType != expectedPathType {
					t.Errorf("Match %d: expected PathType %s, got %s (matches returned in wrong order)",
						i, expectedPathType, match.PathType)
				}

				// Verify Results exists
				if match.Results == nil {
					t.Errorf("Match %d: Results is nil", i)
					continue
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

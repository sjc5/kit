package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

/////////////////////////////////////////////////////////////////////////////
/////////// NESTED ROUTING SCENARIOS ////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////

// TestRouteScenario defines test scenarios for FindAllMatches
type TestRouteScenario struct {
	Path            string
	ExpectedMatches ExpectedMatches
}

type ExpectedMatch struct {
	Pattern           string
	ExpectedParams    Params
	ExpectedSplatSegs []string
	PathType          LastSegmentType // Moved PathType here
}

type ExpectedMatches []ExpectedMatch

// RouteScenarios contains all the test scenarios adapted from matcher.PathScenarios
var RouteScenarios = []TestRouteScenario{
	{
		Path: "/does-not-exist",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:           "/$",
				PathType:          LastSegmentTypes.Splat, // UltimateCatch → Splat
				ExpectedSplatSegs: []string{"does-not-exist"},
			},
		},
	},
	{
		Path: "/this-should-be-ignored",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:           "/$",
				PathType:          LastSegmentTypes.Splat, // UltimateCatch → Splat
				ExpectedSplatSegs: []string{"this-should-be-ignored"},
			},
		},
	},
	{
		Path: "/",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/lion",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/lion",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/lion",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/lion/123",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/lion",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:           "/lion/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedSplatSegs: []string{"123"},
			},
		},
	},
	{
		Path: "/lion/123/456",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/lion",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:           "/lion/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedSplatSegs: []string{"123", "456"},
			},
		},
	},
	{
		Path: "/lion/123/456/789",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/lion",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:           "/lion/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedSplatSegs: []string{"123", "456", "789"},
			},
		},
	},
	{
		Path: "/tiger",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/tiger",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/tiger",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/tiger/123",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/tiger",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/tiger/$tiger_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"tiger_id": "123"},
			},
			{
				Pattern:        "/tiger/$tiger_id",
				PathType:       LastSegmentTypes.Index,
				ExpectedParams: Params{"tiger_id": "123"},
			},
		},
	},
	{
		Path: "/tiger/123/456",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/tiger",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/tiger/$tiger_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"tiger_id": "123"},
			},
			{
				Pattern:        "/tiger/$tiger_id/$tiger_cub_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"tiger_id": "123", "tiger_cub_id": "456"},
			},
		},
	},
	{
		Path: "/tiger/123/456/789",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/tiger",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/tiger/$tiger_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"tiger_id": "123"},
			},
			{
				Pattern:           "/tiger/$tiger_id/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedParams:    Params{"tiger_id": "123"},
				ExpectedSplatSegs: []string{"456", "789"},
			},
		},
	},
	{
		Path: "/bear",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/bear",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/bear",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/bear/123",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/bear",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/bear/$bear_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"bear_id": "123"},
			},
		},
	},
	{
		Path: "/bear/123/456",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/bear",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/bear/$bear_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"bear_id": "123"},
			},
			{
				Pattern:           "/bear/$bear_id/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedParams:    Params{"bear_id": "123"},
				ExpectedSplatSegs: []string{"456"},
			},
		},
	},
	{
		Path: "/bear/123/456/789",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/bear",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/bear/$bear_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"bear_id": "123"},
			},
			{
				Pattern:           "/bear/$bear_id/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedParams:    Params{"bear_id": "123"},
				ExpectedSplatSegs: []string{"456", "789"},
			},
		},
	},
	{
		Path: "/dashboard",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/dashboard/asdf",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:           "/dashboard/$",
				PathType:          LastSegmentTypes.Splat, // NonUltimateSplat → Splat
				ExpectedSplatSegs: []string{"asdf"},
			},
		},
	},
	{
		Path: "/dashboard/customers",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard/customers",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard/customers",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/dashboard/customers/123",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard/customers",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/dashboard/customers/$customer_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"customer_id": "123"},
			},
			{
				Pattern:        "/dashboard/customers/$customer_id",
				PathType:       LastSegmentTypes.Index,
				ExpectedParams: Params{"customer_id": "123"},
			},
		},
	},
	{
		Path: "/dashboard/customers/123/orders",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard/customers",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/dashboard/customers/$customer_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"customer_id": "123"},
			},
			{
				Pattern:        "/dashboard/customers/$customer_id/orders",
				PathType:       LastSegmentTypes.Static, // StaticLayout → Static
				ExpectedParams: Params{"customer_id": "123"},
			},
			{
				Pattern:        "/dashboard/customers/$customer_id/orders",
				PathType:       LastSegmentTypes.Index,
				ExpectedParams: Params{"customer_id": "123"},
			},
		},
	},
	{
		Path: "/dashboard/customers/123/orders/456",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dashboard",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:  "/dashboard/customers",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
			{
				Pattern:        "/dashboard/customers/$customer_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"customer_id": "123"},
			},
			{
				Pattern:        "/dashboard/customers/$customer_id/orders",
				PathType:       LastSegmentTypes.Static, // StaticLayout → Static
				ExpectedParams: Params{"customer_id": "123"},
			},
			{
				Pattern:        "/dashboard/customers/$customer_id/orders/$order_id",
				PathType:       LastSegmentTypes.Dynamic, // DynamicLayout → Dynamic
				ExpectedParams: Params{"customer_id": "123", "order_id": "456"},
			},
		},
	},
	{
		Path: "/articles",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/articles",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/articles/bob",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:           "/$",
				PathType:          LastSegmentTypes.Splat, // UltimateCatch → Splat
				ExpectedSplatSegs: []string{"articles", "bob"},
			},
		},
	},
	{
		Path: "/articles/test",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:           "/$",
				PathType:          LastSegmentTypes.Splat, // UltimateCatch → Splat
				ExpectedSplatSegs: []string{"articles", "test"},
			},
		},
	},
	{
		Path: "/articles/test/articles",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/articles/test/articles",
				PathType: LastSegmentTypes.Index,
			},
		},
	},
	{
		Path: "/dynamic-index/index",
		ExpectedMatches: ExpectedMatches{
			{
				Pattern:  "/dynamic-index/index",
				PathType: LastSegmentTypes.Static, // StaticLayout → Static
			},
		},
	},
}

// func TestNestedRouter_FindAllMatches(t *testing.T) {
// 	// Initialize router with all patterns from test cases
// 	r := NewRouter()
// 	patterns := []struct {
// 		pattern string
// 		isIndex bool
// 	}{
// 		{"/", true},                                        // Index
// 		{"/articles", true},                                // Index
// 		{"/articles/test/articles", true},                  // Index
// 		{"/bear", true},                                    // Index
// 		{"/dashboard", true},                               // Index
// 		{"/dashboard/customers", true},                     // Index
// 		{"/dashboard/customers/$customer_id", true},        // Index
// 		{"/dashboard/customers/$customer_id/orders", true}, // Index
// 		{"/dynamic-index/$pagename", true},                 // Index
// 		{"/lion", true},                                    // Index
// 		{"/tiger", true},                                   // Index
// 		{"/tiger/$tiger_id", true},                         // Index

// 		{"/$", false},
// 		{"/bear", false},
// 		{"/bear/$bear_id", false},
// 		{"/bear/$bear_id/$", false},
// 		{"/dashboard", false},
// 		{"/dashboard/$", false},
// 		{"/dashboard/customers", false},
// 		{"/dashboard/customers/$customer_id", false},
// 		{"/dashboard/customers/$customer_id/orders", false},
// 		{"/dashboard/customers/$customer_id/orders/$order_id", false},
// 		{"/dynamic-index/index", false},
// 		{"/lion", false},
// 		{"/lion/$", false},
// 		{"/tiger", false},
// 		{"/tiger/$tiger_id", false},
// 		{"/tiger/$tiger_id/$tiger_cub_id", false},
// 		{"/tiger/$tiger_id/$", false},
// 	}
// 	for _, p := range patterns {
// 		r.AddRouteWithSegments(ParseSegments(p.pattern), p.isIndex)
// 	}

// 	for _, tc := range RouteScenarios {
// 		t.Run(tc.Path, func(t *testing.T) {
// 			segments := ParseSegments(tc.Path)
// 			actualMatches, ok := r.FindAllMatches(segments)

// 			var errors []string

// 			// Check if there's a failure
// 			expectedCount := len(tc.ExpectedMatches)
// 			actualCount := len(actualMatches)

// 			fail := !ok && expectedCount > 0 || expectedCount != actualCount
// 			for i := 0; i < max(expectedCount, actualCount); i++ {
// 				if i < expectedCount && i < actualCount {
// 					expected := tc.ExpectedMatches[i]
// 					actual := actualMatches[i]

// 					if expected.Pattern != actual.Pattern ||
// 						!reflect.DeepEqual(expected.ExpectedParams, actual.Params) ||
// 						!reflect.DeepEqual(expected.ExpectedSplatSegs, actual.SplatSegments) ||
// 						string(expected.PathType) != string(actual.LastSegmentType) {
// 						fail = true
// 						break
// 					}
// 				} else {
// 					fail = true
// 					break
// 				}
// 			}

// 			// Only output errors if a failure occurred
// 			if fail {
// 				errors = append(errors, fmt.Sprintf("\n===== Path: %q =====", tc.Path))

// 				// Expected matches exist but got none
// 				if !ok && expectedCount > 0 {
// 					errors = append(errors, "Expected matches but got none.")
// 				}

// 				// Length mismatch
// 				if expectedCount != actualCount {
// 					errors = append(errors, fmt.Sprintf("Expected %d matches, got %d", expectedCount, actualCount))
// 				}

// 				// Always output all expected and actual matches for debugging
// 				errors = append(errors, "Expected Matches:")
// 				for i, expected := range tc.ExpectedMatches {
// 					errors = append(errors, fmt.Sprintf(
// 						"  [%d] {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
// 						i, expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
// 					))
// 				}

// 				errors = append(errors, "Actual Matches:")
// 				for i, actual := range actualMatches {
// 					errors = append(errors, fmt.Sprintf(
// 						"  [%d] {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
// 						i, actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
// 					))
// 				}

// 				// Detailed mismatches
// 				for i := 0; i < max(expectedCount, actualCount); i++ {
// 					if i < expectedCount && i < actualCount {
// 						expected := tc.ExpectedMatches[i]
// 						actual := actualMatches[i]

// 						if expected.Pattern != actual.Pattern ||
// 							!reflect.DeepEqual(expected.ExpectedParams, actual.Params) ||
// 							!reflect.DeepEqual(expected.ExpectedSplatSegs, actual.SplatSegments) ||
// 							string(expected.PathType) != string(actual.LastSegmentType) {
// 							errors = append(errors, fmt.Sprintf(
// 								"Match %d mismatch:\n  Expected: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}\n  Got:      {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
// 								i,
// 								expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
// 								actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
// 							))
// 						}
// 					} else if i < expectedCount {
// 						// Missing expected match
// 						expected := tc.ExpectedMatches[i]
// 						errors = append(errors, fmt.Sprintf(
// 							"Missing expected match %d: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
// 							i, expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
// 						))
// 					} else {
// 						// Unexpected extra match
// 						actual := actualMatches[i]
// 						errors = append(errors, fmt.Sprintf(
// 							"Unexpected extra match %d: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
// 							i, actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
// 						))
// 					}
// 				}

// 				// Print only if something went wrong
// 				t.Error(strings.Join(errors, "\n"))
// 			}
// 		})
// 	}
// }

func TestNestedRouter_FindAllMatches(t *testing.T) {
	// Initialize router with all patterns from test cases
	r := NewRouter()
	patterns := []struct {
		pattern string
		isIndex bool
	}{
		{"/", true},                                        // Index
		{"/articles", true},                                // Index
		{"/articles/test/articles", true},                  // Index
		{"/bear", true},                                    // Index
		{"/dashboard", true},                               // Index
		{"/dashboard/customers", true},                     // Index
		{"/dashboard/customers/$customer_id", true},        // Index
		{"/dashboard/customers/$customer_id/orders", true}, // Index
		{"/dynamic-index/$pagename", true},                 // Index
		{"/lion", true},                                    // Index
		{"/tiger", true},                                   // Index
		{"/tiger/$tiger_id", true},                         // Index

		{"/$", false},
		{"/bear", false},
		{"/bear/$bear_id", false},
		{"/bear/$bear_id/$", false},
		{"/dashboard", false},
		{"/dashboard/$", false},
		{"/dashboard/customers", false},
		{"/dashboard/customers/$customer_id", false},
		{"/dashboard/customers/$customer_id/orders", false},
		{"/dashboard/customers/$customer_id/orders/$order_id", false},
		{"/dynamic-index/index", false},
		{"/lion", false},
		{"/lion/$", false},
		{"/tiger", false},
		{"/tiger/$tiger_id", false},
		{"/tiger/$tiger_id/$tiger_cub_id", false},
		{"/tiger/$tiger_id/$", false},
	}
	for _, p := range patterns {
		r.AddRouteWithSegments(ParseSegments(p.pattern), p.isIndex)
	}

	for _, tc := range RouteScenarios {
		t.Run(tc.Path, func(t *testing.T) {
			segments := ParseSegments(tc.Path)
			actualMatches, ok := r.FindAllMatches(segments)

			var errors []string

			// Check if there's a failure
			expectedCount := len(tc.ExpectedMatches)
			actualCount := len(actualMatches)

			fail := (!ok && expectedCount > 0) || (expectedCount != actualCount)

			// Compare each matched route
			for i := 0; i < max(expectedCount, actualCount); i++ {
				if i < expectedCount && i < actualCount {
					expected := tc.ExpectedMatches[i]
					actual := actualMatches[i]

					// ---- Use helper functions to compare maps/slices ----
					if expected.Pattern != actual.Pattern ||
						!equalParams(expected.ExpectedParams, actual.Params) ||
						!equalSplat(expected.ExpectedSplatSegs, actual.SplatSegments) ||
						string(expected.PathType) != string(actual.LastSegmentType) {
						fail = true
						break
					}
				} else {
					fail = true
					break
				}
			}

			// Only output errors if a failure occurred
			if fail {
				errors = append(errors, fmt.Sprintf("\n===== Path: %q =====", tc.Path))

				// Expected matches exist but got none
				if !ok && expectedCount > 0 {
					errors = append(errors, "Expected matches but got none.")
				}

				// Length mismatch
				if expectedCount != actualCount {
					errors = append(errors, fmt.Sprintf("Expected %d matches, got %d", expectedCount, actualCount))
				}

				// Always output all expected and actual matches for debugging
				errors = append(errors, "Expected Matches:")
				for i, expected := range tc.ExpectedMatches {
					errors = append(errors, fmt.Sprintf(
						"  [%d] {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
						i, expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
					))
				}

				errors = append(errors, "Actual Matches:")
				for i, actual := range actualMatches {
					errors = append(errors, fmt.Sprintf(
						"  [%d] {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
						i, actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
					))
				}

				// Detailed mismatches
				for i := 0; i < max(expectedCount, actualCount); i++ {
					if i < expectedCount && i < actualCount {
						expected := tc.ExpectedMatches[i]
						actual := actualMatches[i]

						if expected.Pattern != actual.Pattern ||
							!equalParams(expected.ExpectedParams, actual.Params) ||
							!equalSplat(expected.ExpectedSplatSegs, actual.SplatSegments) ||
							string(expected.PathType) != string(actual.LastSegmentType) {
							errors = append(errors, fmt.Sprintf(
								"Match %d mismatch:\n  Expected: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}\n  Got:      {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
								i,
								expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
								actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
							))
						}
					} else if i < expectedCount {
						// Missing expected match
						expected := tc.ExpectedMatches[i]
						errors = append(errors, fmt.Sprintf(
							"Missing expected match %d: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
							i, expected.Pattern, expected.ExpectedParams, expected.ExpectedSplatSegs, string(expected.PathType),
						))
					} else {
						// Unexpected extra match
						actual := actualMatches[i]
						errors = append(errors, fmt.Sprintf(
							"Unexpected extra match %d: {Pattern: %q, Params: %v, SplatSegments: %v, LastSegmentType: %q}",
							i, actual.Pattern, actual.Params, actual.SplatSegments, actual.LastSegmentType,
						))
					}
				}

				// Print only if something went wrong
				t.Error(strings.Join(errors, "\n"))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helper functions to treat nil maps/slices as empty, avoiding false mismatches
// ---------------------------------------------------------------------------

func equalParams(a, b Params) bool {
	// Consider nil and empty as the same
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func equalSplat(a, b []string) bool {
	// Consider nil and empty slice as the same
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

/////////////////////////////////////////////////////////////////////////////
/////////// SIMPLE ROUTING SCENARIOS ////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////

func TestSimpleRouter_FindBestMatch(t *testing.T) {
	tests := []struct {
		name              string
		routes            []string
		path              string
		wantMatch         bool
		wantPattern       string
		wantParams        Params
		wantSplatSegments []string
	}{
		// index
		{
			name:              "root path",
			routes:            []string{"/", "/$"},
			path:              "/",
			wantMatch:         true,
			wantPattern:       "/",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "exact match",
			routes:            []string{"/", "/users", "/posts"},
			path:              "/users",
			wantMatch:         true,
			wantPattern:       "/users",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "parameter match",
			routes:            []string{"/users", "/users/$id", "/users/profile"},
			path:              "/users/123",
			wantMatch:         true,
			wantPattern:       "/users/$id",
			wantParams:        Params{"id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "multiple matches",
			routes:            []string{"/", "/api", "/api/$version", "/api/v1"},
			path:              "/api/v1",
			wantMatch:         true,
			wantPattern:       "/api/v1",
			wantParams:        nil,
			wantSplatSegments: nil,
		},
		{
			name:              "splat match",
			routes:            []string{"/files", "/files/$"},
			path:              "/files/documents/report.pdf",
			wantMatch:         true,
			wantPattern:       "/files/$",
			wantParams:        nil,
			wantSplatSegments: []string{"documents", "report.pdf"},
		},
		{
			name:              "no match",
			routes:            []string{"/users", "/posts", "/settings"},
			path:              "/profile",
			wantMatch:         false,
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
			wantMatch:         true,
			wantPattern:       "/api/$version/users/$id/posts",
			wantParams:        Params{"version": "v2", "id": "123"},
			wantSplatSegments: nil,
		},
		{
			name:              "empty routes",
			routes:            []string{},
			path:              "/users",
			wantMatch:         false,
			wantPattern:       "",
			wantParams:        nil,
			wantSplatSegments: nil,
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

			// Compare splat segments
			if !reflect.DeepEqual(match.SplatSegments, tt.wantSplatSegments) {
				t.Errorf("FindBestMatch() splat segments = %v, want %v", match.SplatSegments, tt.wantSplatSegments)
			}
		})
	}
}

func TestSimpleRouter_ManyParams(t *testing.T) {
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

func TestSimpleRouter_NestedNoMatch(t *testing.T) {
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

/////////////////////////////////////////////////////////////////////////////
/////////// UTILITIES ///////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////

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

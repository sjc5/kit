package router

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var NestedPatterns = []string{
	"/_index",                                         // Index
	"/articles/_index",                                // Index
	"/articles/test/articles/_index",                  // Index
	"/bear/_index",                                    // Index
	"/dashboard/_index",                               // Index
	"/dashboard/customers/_index",                     // Index
	"/dashboard/customers/$customer_id/_index",        // Index
	"/dashboard/customers/$customer_id/orders/_index", // Index
	"/dynamic-index/$pagename/_index",                 // Index
	"/lion/_index",                                    // Index
	"/tiger/_index",                                   // Index
	"/tiger/$tiger_id/_index",                         // Index

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

// TestRouteScenario defines test scenarios for FindAllMatches
type TestRouteScenario struct {
	Path            string
	ExpectedMatches []Match
}

// RouteScenarios contains all the test scenarios adapted from matcher.PathScenarios
var RouteScenarios = []TestRouteScenario{
	{
		Path: "/does-not-exist",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/$"},
				SplatValues:     []string{"does-not-exist"},
			},
		},
	},
	{
		Path: "/this-should-be-ignored",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/$"},
				SplatValues:     []string{"this-should-be-ignored"},
			},
		},
	},
	{
		Path: "/",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/_index"},
			},
		},
	},
	{
		Path: "/lion",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion/_index"},
			},
		},
	},
	{
		Path: "/lion/123",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion/$"},
				SplatValues:     []string{"123"},
			},
		},
	},
	{
		Path: "/lion/123/456",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion/$"},
				SplatValues:     []string{"123", "456"},
			},
		},
	},
	{
		Path: "/lion/123/456/789",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/lion/$"},
				SplatValues:     []string{"123", "456", "789"},
			},
		},
	},
	{
		Path: "/tiger",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/_index"},
			},
		},
	},
	{
		Path: "/tiger/123",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id"},
				Params:          Params{"tiger_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id/_index"},
				Params:          Params{"tiger_id": "123"},
			},
		},
	},
	{
		Path: "/tiger/123/456",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id"},
				Params:          Params{"tiger_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id/$tiger_cub_id"},
				Params:          Params{"tiger_id": "123", "tiger_cub_id": "456"},
			},
		},
	},
	{
		Path: "/tiger/123/456/789",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id"},
				Params:          Params{"tiger_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/tiger/$tiger_id/$"},
				Params:          Params{"tiger_id": "123"},
				SplatValues:     []string{"456", "789"},
			},
		},
	},
	{
		Path: "/bear",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/_index"},
			},
		},
	},
	{
		Path: "/bear/123",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/$bear_id"},
				Params:          Params{"bear_id": "123"},
			},
		},
	},
	{
		Path: "/bear/123/456",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/$bear_id"},
				Params:          Params{"bear_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/$bear_id/$"},
				Params:          Params{"bear_id": "123"},
				SplatValues:     []string{"456"},
			},
		},
	},
	{
		Path: "/bear/123/456/789",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/$bear_id"},
				Params:          Params{"bear_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/bear/$bear_id/$"},
				Params:          Params{"bear_id": "123"},
				SplatValues:     []string{"456", "789"},
			},
		},
	},
	{
		Path: "/dashboard",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/_index"},
			},
		},
	},
	{
		Path: "/dashboard/asdf",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/$"},
				SplatValues:     []string{"asdf"},
			},
		},
	},
	{
		Path: "/dashboard/customers",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/_index"},
			},
		},
	},
	{
		Path: "/dashboard/customers/123",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id"},
				Params:          Params{"customer_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id/_index"},
				Params:          Params{"customer_id": "123"},
			},
		},
	},
	{
		Path: "/dashboard/customers/123/orders",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id"},
				Params:          Params{"customer_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id/orders"},
				Params:          Params{"customer_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id/orders/_index"},
				Params:          Params{"customer_id": "123"},
			},
		},
	},
	{
		Path: "/dashboard/customers/123/orders/456",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id"},
				Params:          Params{"customer_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id/orders"},
				Params:          Params{"customer_id": "123"},
			},
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/dashboard/customers/$customer_id/orders/$order_id"},
				Params:          Params{"customer_id": "123", "order_id": "456"},
			},
		},
	},
	{
		Path: "/articles",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/articles/_index"},
			},
		},
	},
	{
		Path: "/articles/bob",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/$"},
				SplatValues:     []string{"articles", "bob"},
			},
		},
	},
	{
		Path: "/articles/test",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/$"},
				SplatValues:     []string{"articles", "test"},
			},
		},
	},
	{
		Path: "/articles/test/articles",
		ExpectedMatches: []Match{
			{
				RegisteredRoute: &RegisteredRoute{Pattern: "/articles/test/articles/_index"},
			},
		},
	},
	{
		Path: "/dynamic-index/index",
		ExpectedMatches: []Match{
			{
				// no underscore prefix, so not really an index!
				RegisteredRoute: &RegisteredRoute{Pattern: "/dynamic-index/index"},
			},
		},
	},
}

func TestNestedRouter_FindAllMatches(t *testing.T) {
	// Initialize router with all patterns from test cases
	r := RouterBest{}
	r.NestedIndexSignifier = "_index"
	r.ShouldExcludeSegmentFunc = func(segment string) bool {
		return strings.HasPrefix(segment, "__")
	}

	for _, p := range NestedPatterns {
		r.AddRoute(p)
	}

	// r.PrintRouteMaps()

	for _, tc := range RouteScenarios {
		t.Run(tc.Path, func(t *testing.T) {
			actualMatches, ok := r.FindAllMatches(tc.Path)

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
						!equalParams(expected.Params, actual.Params) ||
						!equalSplat(expected.SplatValues, actual.SplatValues) {
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
						"  [%d] {Pattern: %q, Params: %v, SplatValues: %v}",
						i, expected.Pattern, expected.Params, expected.SplatValues,
					))
				}

				errors = append(errors, "Actual Matches:")
				for i, actual := range actualMatches {
					errors = append(errors, fmt.Sprintf(
						"  [%d] {Pattern: %q, Params: %v, SplatValues: %v}",
						i, actual.Pattern, actual.Params, actual.SplatValues,
					))
				}

				// Detailed mismatches
				for i := 0; i < max(expectedCount, actualCount); i++ {
					if i < expectedCount && i < actualCount {
						expected := tc.ExpectedMatches[i]
						actual := actualMatches[i]

						if expected.Pattern != actual.Pattern ||
							!equalParams(expected.Params, actual.Params) ||
							!equalSplat(expected.SplatValues, actual.SplatValues) {
							errors = append(errors, fmt.Sprintf(
								"Match %d mismatch:\n  Expected: {Pattern: %q, Params: %v, SplatValues: %v}\n  Got:      {Pattern: %q, Params: %v, SplatValues: %v}",
								i,
								expected.Pattern, expected.Params, expected.SplatValues,
								actual.Pattern, actual.Params, actual.SplatValues,
							))
						}
					} else if i < expectedCount {
						// Missing expected match
						expected := tc.ExpectedMatches[i]
						errors = append(errors, fmt.Sprintf(
							"Missing expected match %d: {Pattern: %q, Params: %v, SplatValues: %v}",
							i, expected.Pattern, expected.Params, expected.SplatValues,
						))
					} else {
						// Unexpected extra match
						actual := actualMatches[i]
						errors = append(errors, fmt.Sprintf(
							"Unexpected extra match %d: {Pattern: %q, Params: %v, SplatValues: %v}",
							i, actual.Pattern, actual.Params, actual.SplatValues,
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

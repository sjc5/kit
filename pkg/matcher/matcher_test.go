package matcher

import (
	"reflect"
	"testing"
)

// rawPatterns_PreBuild contains all the route patterns to test -- have not been run through PatternToRegisteredPath
var rawPatterns_PreBuild = []string{
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
	"/dynamic-index/__site_index/index",
	"/lion",
	"/lion/$",
	"/tiger",
	"/tiger/$tiger_id",
	"/tiger/$tiger_id/$tiger_cub_id",
	"/tiger/$tiger_id/$",
}

func TestPatternToRegisteredPath(t *testing.T) {
	for i, pattern := range rawPatterns_PreBuild {
		t.Run(pattern, func(t *testing.T) {
			expected := finalRegisteredPathsForTest[i]
			result := PatternToRegisteredPath(pattern)

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("PatternToRegisteredPath(%q)\n got: %+v\nwant: %+v", pattern, result, expected)

				if result.Pattern != expected.Pattern {
					t.Errorf("  Pattern: got %q, want %q", result.Pattern, expected.Pattern)
				}
				if result.PathType != expected.PathType {
					t.Errorf("  PathType: got %v, want %v", result.PathType, expected.PathType)
				}
			}
		})
	}
}

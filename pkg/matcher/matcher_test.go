package matcher

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

// func TestPatternToRegisteredPath(t *testing.T) {
// 	for i, pattern := range rawPatterns_PreBuild {
// 		t.Run(pattern, func(t *testing.T) {
// 			expected := finalRegisteredPathsForTest[i]
// 			result := PatternToRegisteredPath(pattern)

// 			if !reflect.DeepEqual(result, expected) {
// 				t.Errorf("PatternToRegisteredPath(%q)\n got: %+v\nwant: %+v", pattern, result, expected)

// 				if result.Pattern != expected.Pattern {
// 					t.Errorf("  Pattern: got %q, want %q", result.Pattern, expected.Pattern)
// 				}
// 				if !reflect.DeepEqual(result.Segments, expected.Segments) {
// 					t.Errorf("  Segments: got %v, want %v", result.Segments, expected.Segments)
// 				}
// 				if result.PathType != expected.PathType {
// 					t.Errorf("  PathType: got %v, want %v", result.PathType, expected.PathType)
// 				}
// 			}
// 		})
// 	}
// }

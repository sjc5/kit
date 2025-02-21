package matcher

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/sjc5/kit/pkg/router"
)

// finalRegisteredPathsForTest contains the expected RegisteredPath objects for each pattern
var finalRegisteredPathsForTest = []*RegisteredPath{
	{Pattern: "/$", Segments: []string{"$"}, PathType: PathTypes.UltimateCatch},
	{Pattern: "/_index", Segments: []string{"_index"}, PathType: PathTypes.Index},
	{Pattern: "/articles/_index", Segments: []string{"articles", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/articles/test/articles/_index", Segments: []string{"articles", "test", "articles", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/bear/$bear_id/$", Segments: []string{"bear", "$bear_id", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/bear/$bear_id", Segments: []string{"bear", "$bear_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/bear/_index", Segments: []string{"bear", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/bear", Segments: []string{"bear"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/$", Segments: []string{"dashboard", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/dashboard/_index", Segments: []string{"dashboard", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/_index", Segments: []string{"dashboard", "customers", "$customer_id", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/orders/$order_id", Segments: []string{"dashboard", "customers", "$customer_id", "orders", "$order_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/dashboard/customers/$customer_id/orders/_index", Segments: []string{"dashboard", "customers", "$customer_id", "orders", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers/$customer_id/orders", Segments: []string{"dashboard", "customers", "$customer_id", "orders"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard/customers/$customer_id", Segments: []string{"dashboard", "customers", "$customer_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/dashboard/customers/_index", Segments: []string{"dashboard", "customers", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/dashboard/customers", Segments: []string{"dashboard", "customers"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dashboard", Segments: []string{"dashboard"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/dynamic-index/$pagename/_index", Segments: []string{"dynamic-index", "$pagename", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/dynamic-index/index", Segments: []string{"dynamic-index", "index"}, PathType: PathTypes.StaticLayout}, // PatternToRegisteredPath strips out segments starting with double underscores
	{Pattern: "/lion/$", Segments: []string{"lion", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/lion/_index", Segments: []string{"lion", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/lion", Segments: []string{"lion"}, PathType: PathTypes.StaticLayout},
	{Pattern: "/tiger/$tiger_id/$", Segments: []string{"tiger", "$tiger_id", "$"}, PathType: PathTypes.NonUltimateSplat},
	{Pattern: "/tiger/$tiger_id/$tiger_cub_id", Segments: []string{"tiger", "$tiger_id", "$tiger_cub_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/$tiger_id/_index", Segments: []string{"tiger", "$tiger_id", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/tiger/$tiger_id", Segments: []string{"tiger", "$tiger_id"}, PathType: PathTypes.DynamicLayout},
	{Pattern: "/tiger/_index", Segments: []string{"tiger", "_index"}, PathType: PathTypes.Index},
	{Pattern: "/tiger", Segments: []string{"tiger"}, PathType: PathTypes.StaticLayout},
}

// IncomingPathsTestCase defines a test case for incoming path matching.
type IncomingPathsTestCase struct {
	Path    string   `json:"path"`
	Matches []*Match `json:"matches"`
}

var incomingPathsTestCases []IncomingPathsTestCase

func init() {
	if err := json.Unmarshal([]byte(incomingPathsJSON), &incomingPathsTestCases); err != nil {
		panic("Failed to parse incoming paths test data: " + err.Error())
	}
}

// TestGetIncomingPaths tests the getIncomingPaths function against predefined test cases.
func TestGetIncomingPaths(t *testing.T) {
	for _, tc := range incomingPathsTestCases {
		t.Run(tc.Path, func(t *testing.T) {
			// Parse the input path into segments
			segments := router.ParseSegments(tc.Path)

			// Get actual matches from the function under test
			gotMatches := getIncomingPaths(finalRegisteredPathsForTest, segments)

			// Check if the number of matches is correct
			if len(gotMatches) != len(tc.Matches) {
				t.Errorf("expected %d matches, got %d", len(tc.Matches), len(gotMatches))
				logMatchDiff(t, tc.Matches, gotMatches)
				return
			}

			// Compare each match individually
			for i, expected := range tc.Matches {
				if i >= len(gotMatches) {
					t.Errorf("match %d missing in actual results", i)
					continue
				}
				got := gotMatches[i]
				compareMatch(t, tc.Path, i, expected, got)
			}
		})
	}
}

// logMatchDiff logs the difference between expected and actual matches when lengths differ.
func logMatchDiff(t *testing.T, expected, got []*Match) {
	t.Log("Expected matches:")
	for i, m := range expected {
		t.Logf("  %d: %s", i, formatMatch(m))
	}
	t.Log("Got matches:")
	for i, m := range got {
		t.Logf("  %d: %s", i, formatMatch(m))
	}
}

// compareMatch compares an expected match with an actual match and reports differences.
func compareMatch(t *testing.T, path string, index int, expected, got *Match) {
	if !reflect.DeepEqual(expected.RegisteredPath, got.RegisteredPath) {
		t.Errorf("match %d for path %q: RegisteredPath mismatch\n  expected: %v\n  got:      %v",
			index, path, *expected.RegisteredPath, *got.RegisteredPath)
	}
	if !reflect.DeepEqual(expected.Results, got.Results) {
		t.Errorf("match %d for path %q: Results mismatch\n  expected: %v\n  got:      %v",
			index, path, *expected.Results, *got.Results)
	}
}

// formatMatch returns a concise string representation of a Match for logging.
func formatMatch(m *Match) string {
	if m.RegisteredPath == nil || m.Results == nil {
		return "Match{RegisteredPath:nil Results:nil}"
	}
	return fmt.Sprintf("Match{Path:%s Params:%v Splat:%v Score:%d}",
		m.RegisteredPath.Pattern, m.Results.Params, m.Results.SplatSegments, m.Results.Score)
}

// // Helper struct to define expected test cases
// type IncomingPathsTestCase struct {
// 	Path    string  `json:"path"`
// 	Matches []*Match `json:"matches"`
// }

// func GenerateIncomingPathsTestData() {
// 	var results []IncomingPathsTestCase
// 	for _, tc := range PathScenarios {
// 		realSegments := router.ParseSegments(tc.Path)
// 		incomingPaths := getIncomingPaths(finalRegisteredPathsForTest, realSegments)
// 		var matches []*Match

// 		for _, match := range incomingPaths {
// 			matches = append(matches, Match{
// 				RegisteredPath: &RegisteredPath{
// 					Pattern:  match.Pattern,
// 					Segments: match.Segments,
// 					PathType: match.PathType,
// 				},
// 				Results: &Results{
// 					Params:             match.Results.Params,
// 					SplatSegments:      match.Results.SplatSegments,
// 					Score:              match.Results.Score,
// 					RealSegmentsLength: match.Results.RealSegmentsLength,
// 				},
// 			})
// 		}

// 		results = append(results, IncomingPathsTestCase{
// 			Path:    tc.Path,
// 			Matches: matches,
// 		})
// 	}

// 	jsonData, _ := json.Marshal(results)
// 	fmt.Println(string(jsonData))
// }

// // Run this function manually to generate expected output for tests
// func TestGenerateIncomingPathsTestData(t *testing.T) {
// 	GenerateIncomingPathsTestData()
// }

const incomingPathsJSON = `[{"path":"/does-not-exist","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["does-not-exist"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1}]},{"path":"/this-should-be-ignored","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["this-should-be-ignored"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1}]},{"path":"/","matches":[{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":0}]},{"path":"/lion","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/lion/_index","segments":["lion","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/lion/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123"],"Score":4,"RealSegmentsLength":2},{"pattern":"/lion/_index","segments":["lion","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/lion/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123","456"],"Score":4,"RealSegmentsLength":3},{"pattern":"/lion/_index","segments":["lion","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/lion/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123","456","789"],"Score":4,"RealSegmentsLength":4},{"pattern":"/lion/_index","segments":["lion","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/tiger","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/tiger/_index","segments":["tiger","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/tiger/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/tiger/$tiger_id/_index","segments":["tiger","$tiger_id","_index"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/tiger/_index","segments":["tiger","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/tiger/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id/$","segments":["tiger","$tiger_id","$"],"routeType":"ends-in-splat","Params":{"tiger_id":"123"},"SplatSegments":["456"],"Score":6,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id/$tiger_cub_id","segments":["tiger","$tiger_id","$tiger_cub_id"],"routeType":"dynamic","Params":{"tiger_cub_id":"456","tiger_id":"123"},"SplatSegments":null,"Score":7,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id/_index","segments":["tiger","$tiger_id","_index"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/tiger/_index","segments":["tiger","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/tiger/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id/$","segments":["tiger","$tiger_id","$"],"routeType":"ends-in-splat","Params":{"tiger_id":"123"},"SplatSegments":["456","789"],"Score":6,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id/$tiger_cub_id","segments":["tiger","$tiger_id","$tiger_cub_id"],"routeType":"dynamic","Params":{"tiger_cub_id":"456","tiger_id":"123"},"SplatSegments":null,"Score":7,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id/_index","segments":["tiger","$tiger_id","_index"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/tiger/_index","segments":["tiger","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/bear","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/bear/_index","segments":["bear","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/bear/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/bear/_index","segments":["bear","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/bear/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/bear/$bear_id/$","segments":["bear","$bear_id","$"],"routeType":"ends-in-splat","Params":{"bear_id":"123"},"SplatSegments":["456"],"Score":6,"RealSegmentsLength":3},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/bear/_index","segments":["bear","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/bear/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/bear/$bear_id/$","segments":["bear","$bear_id","$"],"routeType":"ends-in-splat","Params":{"bear_id":"123"},"SplatSegments":["456","789"],"Score":6,"RealSegmentsLength":4},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/bear/_index","segments":["bear","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/dashboard","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/dashboard/asdf","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","asdf"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["asdf"],"Score":4,"RealSegmentsLength":2},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/dashboard/customers","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers"],"Score":4,"RealSegmentsLength":2},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/dashboard/customers/_index","segments":["dashboard","customers","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/dashboard/customers/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123"],"Score":1,"RealSegmentsLength":3},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123"],"Score":4,"RealSegmentsLength":3},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/dashboard/customers/$customer_id/_index","segments":["dashboard","customers","$customer_id","_index"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":3},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":3},{"pattern":"/dashboard/customers/_index","segments":["dashboard","customers","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":3},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":3},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/dashboard/customers/123/orders","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123","orders"],"Score":1,"RealSegmentsLength":4},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123","orders"],"Score":4,"RealSegmentsLength":4},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id/_index","segments":["dashboard","customers","$customer_id","_index"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id/orders/_index","segments":["dashboard","customers","$customer_id","orders","_index"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"static","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/_index","segments":["dashboard","customers","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":4},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":4},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/dashboard/customers/123/orders/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123","orders","456"],"Score":1,"RealSegmentsLength":5},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":5},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123","orders","456"],"Score":4,"RealSegmentsLength":5},{"pattern":"/dashboard/_index","segments":["dashboard","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/_index","segments":["dashboard","customers","$customer_id","_index"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders/$order_id","segments":["dashboard","customers","$customer_id","orders","$order_id"],"routeType":"dynamic","Params":{"customer_id":"123","order_id":"456"},"SplatSegments":null,"Score":13,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders/_index","segments":["dashboard","customers","$customer_id","orders","_index"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"static","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/_index","segments":["dashboard","customers","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":5},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":5},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":5}]},{"path":"/articles","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles"],"Score":1,"RealSegmentsLength":1},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/articles/_index","segments":["articles","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/articles/bob","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","bob"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/articles/_index","segments":["articles","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/articles/test","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","test"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/articles/_index","segments":["articles","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/articles/test/articles","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","test","articles"],"Score":1,"RealSegmentsLength":3},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/articles/_index","segments":["articles","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/articles/test/articles/_index","segments":["articles","test","articles","_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":9,"RealSegmentsLength":3}]},{"path":"/dynamic-index/index","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dynamic-index","index"],"Score":1,"RealSegmentsLength":2},{"pattern":"/_index","segments":["_index"],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dynamic-index/$pagename/_index","segments":["dynamic-index","$pagename","_index"],"routeType":"index","Params":{"pagename":"index"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/dynamic-index/index","segments":["dynamic-index","index"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2}]}]`

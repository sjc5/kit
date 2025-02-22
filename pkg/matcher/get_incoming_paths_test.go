package matcher

// // IncomingPathsTestCase defines a test case for incoming path matching.
// type IncomingPathsTestCase struct {
// 	Path    string   `json:"path"`
// 	Matches []*Match `json:"matches"`
// }

// var incomingPathsTestCases []IncomingPathsTestCase

// func init() {
// 	if err := json.Unmarshal([]byte(incomingPathsJSON), &incomingPathsTestCases); err != nil {
// 		panic("Failed to parse incoming paths test data: " + err.Error())
// 	}

// 	// remove all matches with a score of zero
// 	for i, tc := range incomingPathsTestCases {
// 		var matches []*Match
// 		for _, match := range tc.Matches {
// 			if match.Results.Score != 0 {
// 				matches = append(matches, match)
// 			}
// 		}
// 		incomingPathsTestCases[i].Matches = matches
// 	}
// }

// // TestGetIncomingPaths tests the getIncomingPaths function against predefined test cases.
// func TestGetIncomingPaths(t *testing.T) {
// 	for i, tc := range incomingPathsTestCases {
// 		t.Run(tc.Path, func(t *testing.T) {
// 			// Parse the input path into segments
// 			segments := router.ParseSegments(tc.Path)

// 			// Get actual matches from the function under test
// 			gotMatches := getIncomingPaths(finalRegisteredPathsForTest, segments)

// 			// Check if the number of matches is correct
// 			if len(gotMatches) != len(tc.Matches) {
// 				t.Errorf("Path: %q, Index: %d, Expected %d matches, got %d", tc.Path, i, len(tc.Matches), len(gotMatches))
// 				logMatchDiff(t, tc.Matches, gotMatches)
// 				return
// 			}

// 			// Compare each match individually
// 			for i, expected := range tc.Matches {
// 				if i >= len(gotMatches) {
// 					t.Errorf("match %d missing in actual results", i)
// 					continue
// 				}
// 				got := gotMatches[i]
// 				compareMatch(t, tc.Path, i, expected, got)
// 			}
// 		})
// 	}
// }

// // logMatchDiff logs the difference between expected and actual matches when lengths differ.
// func logMatchDiff(t *testing.T, expected, got []*Match) {
// 	t.Log("Expected matches:")
// 	for i, m := range expected {
// 		t.Logf("  %d: %s", i, formatMatch(m))
// 	}
// 	t.Log("Got matches:")
// 	for i, m := range got {
// 		t.Logf("  %d: %s", i, formatMatch(m))
// 	}
// }

// // compareMatch compares an expected match with an actual match and reports differences.
// func compareMatch(t *testing.T, path string, index int, expected, got *Match) {
// 	if !reflect.DeepEqual(expected.RegisteredPath, got.RegisteredPath) {
// 		t.Errorf("match %d for path %q: RegisteredPath mismatch\n  expected: %v\n  got:      %v",
// 			index, path, *expected.RegisteredPath, *got.RegisteredPath)
// 	}
// 	if got.Results.Params == nil {
// 		got.Results.Params = make(Params)
// 	}
// 	if expected.Results.Params != nil && !reflect.DeepEqual(expected.Results.Params, got.Results.Params) {
// 		t.Errorf("match %d for path %q: Params mismatch\n  expected: %v\n  got:      %v",
// 			index, path, expected.Results.Params, got.Results.Params)
// 	}
// 	if expected.Results.SplatSegments != nil && !reflect.DeepEqual(expected.Results.SplatSegments, got.Results.SplatSegments) {
// 		t.Errorf("match %d for path %q: SplatSegments mismatch\n  expected: %v\n  got:      %v",
// 			index, path, expected.Results.SplatSegments, got.Results.SplatSegments)
// 	}
// 	if expected.Results.Score != got.Results.Score {
// 		t.Errorf("match %d for path %q: Score mismatch\n  expected: %d\n  got:      %d",
// 			index, path, expected.Results.Score, got.Results.Score)
// 	}
// 	if expected.Results.RealSegmentsLength != got.Results.RealSegmentsLength {
// 		t.Errorf("match %d for path %q: RealSegmentsLength mismatch\n  expected: %d\n  got:      %d",
// 			index, path, expected.Results.RealSegmentsLength, got.Results.RealSegmentsLength)
// 	}
// }

// // formatMatch returns a concise string representation of a Match for logging.
// func formatMatch(m *Match) string {
// 	if m.RegisteredPath == nil || m.Results == nil {
// 		return "Match{RegisteredPath:nil Results:nil}"
// 	}
// 	return fmt.Sprintf("Match{Path:%s Params:%v Splat:%v Score:%d}",
// 		m.RegisteredPath.Pattern, m.Results.Params, m.Results.SplatSegments, m.Results.Score)
// }

// // // Helper struct to define expected test cases
// // type IncomingPathsTestCase struct {
// // 	Path    string  `json:"path"`
// // 	Matches []*Match `json:"matches"`
// // }

// // func GenerateIncomingPathsTestData() {
// // 	var results []IncomingPathsTestCase
// // 	for _, tc := range PathScenarios {
// // 		realSegments := router.ParseSegments(tc.Path)
// // 		incomingPaths := getIncomingPaths(finalRegisteredPathsForTest, realSegments)
// // 		var matches []*Match

// // 		for _, match := range incomingPaths {
// // 			matches = append(matches, Match{
// // 				RegisteredPath: &RegisteredPath{
// // 					Pattern:  match.Pattern,
// // 					Segments: match.Segments,
// // 					PathType: match.PathType,
// // 				},
// // 				Results: &Results{
// // 					Params:             match.Results.Params,
// // 					SplatSegments:      match.Results.SplatSegments,
// // 					Score:              match.Results.Score,
// // 					RealSegmentsLength: match.Results.RealSegmentsLength,
// // 				},
// // 			})
// // 		}

// // 		results = append(results, IncomingPathsTestCase{
// // 			Path:    tc.Path,
// // 			Matches: matches,
// // 		})
// // 	}

// // 	jsonData, _ := json.Marshal(results)
// // 	fmt.Println(string(jsonData))
// // }

// // // Run this function manually to generate expected output for tests
// // func TestGenerateIncomingPathsTestData(t *testing.T) {
// // 	GenerateIncomingPathsTestData()
// // }

// const incomingPathsJSON = `[{"path":"/does-not-exist","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["does-not-exist"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1}]},{"path":"/this-should-be-ignored","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["this-should-be-ignored"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1}]},{"path":"/","matches":[{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":0}]},{"path":"/lion","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/lion","segments":["lion"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/lion/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123"],"Score":4,"RealSegmentsLength":2},{"pattern":"/lion","segments":["lion"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/lion/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123","456"],"Score":4,"RealSegmentsLength":3},{"pattern":"/lion","segments":["lion"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/lion/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["lion","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/lion/$","segments":["lion","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["123","456","789"],"Score":4,"RealSegmentsLength":4},{"pattern":"/lion","segments":["lion"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/lion","segments":["lion"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/tiger","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/tiger","segments":["tiger"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/tiger/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/tiger","segments":["tiger"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/tiger/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id/$","segments":["tiger","$tiger_id","$"],"routeType":"ends-in-splat","Params":{"tiger_id":"123"},"SplatSegments":["456"],"Score":6,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id/$tiger_cub_id","segments":["tiger","$tiger_id","$tiger_cub_id"],"routeType":"dynamic","Params":{"tiger_cub_id":"456","tiger_id":"123"},"SplatSegments":null,"Score":7,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/tiger","segments":["tiger"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/tiger/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["tiger","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id/$","segments":["tiger","$tiger_id","$"],"routeType":"ends-in-splat","Params":{"tiger_id":"123"},"SplatSegments":["456","789"],"Score":6,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id/$tiger_cub_id","segments":["tiger","$tiger_id","$tiger_cub_id"],"routeType":"dynamic","Params":{"tiger_cub_id":"456","tiger_id":"123"},"SplatSegments":null,"Score":7,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"index","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/tiger/$tiger_id","segments":["tiger","$tiger_id"],"routeType":"dynamic","Params":{"tiger_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/tiger","segments":["tiger"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/tiger","segments":["tiger"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/bear","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/bear","segments":["bear"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/bear/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/bear","segments":["bear"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/bear/123/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123","456"],"Score":1,"RealSegmentsLength":3},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/bear/$bear_id/$","segments":["bear","$bear_id","$"],"routeType":"ends-in-splat","Params":{"bear_id":"123"},"SplatSegments":["456"],"Score":6,"RealSegmentsLength":3},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":3},{"pattern":"/bear","segments":["bear"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/bear/123/456/789","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["bear","123","456","789"],"Score":1,"RealSegmentsLength":4},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/bear/$bear_id/$","segments":["bear","$bear_id","$"],"routeType":"ends-in-splat","Params":{"bear_id":"123"},"SplatSegments":["456","789"],"Score":6,"RealSegmentsLength":4},{"pattern":"/bear/$bear_id","segments":["bear","$bear_id"],"routeType":"dynamic","Params":{"bear_id":"123"},"SplatSegments":null,"Score":5,"RealSegmentsLength":4},{"pattern":"/bear","segments":["bear"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/bear","segments":["bear"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/dashboard","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/dashboard/asdf","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","asdf"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["asdf"],"Score":4,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/dashboard/customers","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers"],"Score":4,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/dashboard/customers/123","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123"],"Score":1,"RealSegmentsLength":3},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123"],"Score":4,"RealSegmentsLength":3},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":3},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":3},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":3},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":3},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3}]},{"path":"/dashboard/customers/123/orders","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123","orders"],"Score":1,"RealSegmentsLength":4},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":4},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123","orders"],"Score":4,"RealSegmentsLength":4},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"static","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":4},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":4},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":4},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":4},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":4}]},{"path":"/dashboard/customers/123/orders/456","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dashboard","customers","123","orders","456"],"Score":1,"RealSegmentsLength":5},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":5},{"pattern":"/dashboard/$","segments":["dashboard","$"],"routeType":"ends-in-splat","Params":{},"SplatSegments":["customers","123","orders","456"],"Score":4,"RealSegmentsLength":5},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders/$order_id","segments":["dashboard","customers","$customer_id","orders","$order_id"],"routeType":"dynamic","Params":{"customer_id":"123","order_id":"456"},"SplatSegments":null,"Score":13,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"index","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id/orders","segments":["dashboard","customers","$customer_id","orders"],"routeType":"static","Params":{"customer_id":"123"},"SplatSegments":null,"Score":11,"RealSegmentsLength":5},{"pattern":"/dashboard/customers/$customer_id","segments":["dashboard","customers","$customer_id"],"routeType":"dynamic","Params":{"customer_id":"123"},"SplatSegments":null,"Score":8,"RealSegmentsLength":5},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"index","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":5},{"pattern":"/dashboard/customers","segments":["dashboard","customers"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":5},{"pattern":"/dashboard","segments":["dashboard"],"routeType":"static","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":5}]},{"path":"/articles","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles"],"Score":1,"RealSegmentsLength":1},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":1},{"pattern":"/articles","segments":["articles"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":1}]},{"path":"/articles/bob","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","bob"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/articles","segments":["articles"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/articles/test","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","test"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/articles","segments":["articles"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":2}]},{"path":"/articles/test/articles","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["articles","test","articles"],"Score":1,"RealSegmentsLength":3},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":3},{"pattern":"/articles","segments":["articles"],"routeType":"index","Params":{},"SplatSegments":null,"Score":3,"RealSegmentsLength":3},{"pattern":"/articles/test/articles","segments":["articles","test","articles"],"routeType":"index","Params":{},"SplatSegments":null,"Score":9,"RealSegmentsLength":3}]},{"path":"/dynamic-index/index","matches":[{"pattern":"/$","segments":["$"],"routeType":"lone-splat","Params":{},"SplatSegments":["dynamic-index","index"],"Score":1,"RealSegmentsLength":2},{"pattern":"/","segments":[],"routeType":"index","Params":{},"SplatSegments":null,"Score":0,"RealSegmentsLength":2},{"pattern":"/dynamic-index/$pagename","segments":["dynamic-index","$pagename"],"routeType":"index","Params":{"pagename":"index"},"SplatSegments":null,"Score":5,"RealSegmentsLength":2},{"pattern":"/dynamic-index/index","segments":["dynamic-index","index"],"routeType":"static","Params":{},"SplatSegments":null,"Score":6,"RealSegmentsLength":2}]}]`

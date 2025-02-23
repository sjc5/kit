package matcher

/////////////////////////////////////////////////////////////////////
/////// PUBLIC API
/////////////////////////////////////////////////////////////////////

// type (
// 	Params          = map[string]string
// 	PathType        string
// 	RegisteredPaths = []*RegisteredPath

// 	Results struct {
// 		Params             Params
// 		SplatSegments      []string
// 		Score              int
// 		RealSegmentsLength int
// 	}

// 	RegisteredPath struct {
// 		Pattern  string   `json:"pattern"`
// 		Segments []string `json:"segments"`
// 		PathType PathType `json:"routeType"`
// 	}

// 	Match struct {
// 		*RegisteredPath
// 		*Results
// 	}
// )

// var (
// 	GetMatchingPaths = getMatchingPathsInternal

// 	PathTypes = struct {
// 		UltimateCatch    PathType // Lone splat segment
// 		NonUltimateSplat PathType // Ends in splat segment
// 		StaticLayout     PathType // Ends in static segment
// 		DynamicLayout    PathType // Ends in dynamic segment
// 		Index            PathType // Ends in index segment (only relevant for nested routing)
// 	}{
// 		UltimateCatch:    "lone-splat",
// 		NonUltimateSplat: "ends-in-splat",
// 		StaticLayout:     "static",
// 		DynamicLayout:    "dynamic",
// 		Index:            "index",
// 	}
// )

/////////////////////////////////////////////////////////////////////
/////// MESSY GET MATCHING PATHS IMPLEMENTATION AND HELPERS
/////////////////////////////////////////////////////////////////////

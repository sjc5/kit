package matcher

import (
	"github.com/sjc5/kit/pkg/router"
)

/////////////////////////////////////////////////////////////////////
/////// PUBLIC API
/////////////////////////////////////////////////////////////////////

type (
	Params          = map[string]string
	PathType        string
	RegisteredPaths = []*RegisteredPath

	Results struct {
		Params             Params
		SplatSegments      []string
		Score              int
		RealSegmentsLength int
	}

	RegisteredPath struct {
		Pattern string `json:"pattern"`
		// Segments []string `json:"segments"`
		PathType PathType `json:"routeType"`
	}

	Match struct {
		*RegisteredPath
		*Results
		Segments []string
	}
)

var (
	GetMatchingPaths = getMatchingPathsInternal

	PathTypes = struct {
		Splat         PathType // Ends in splat segment
		StaticLayout  PathType // Ends in static segment
		DynamicLayout PathType // Ends in dynamic segment
		Index         PathType // Ends in index segment (only relevant for nested routing)
	}{
		Splat:         "Splat",
		StaticLayout:  "Static",
		DynamicLayout: "Dynamic",
		Index:         "Index",
	}
)

/////////////////////////////////////////////////////////////////////
/////// MESSY GET MATCHING PATHS IMPLEMENTATION AND HELPERS
/////////////////////////////////////////////////////////////////////

func getMatchingPathsInternal(registeredPaths RegisteredPaths, realPath string) ([]string, []*Match) {
	realSegments := router.ParseSegments(realPath)
	paths := getIncomingPaths(registeredPaths, realSegments)

	splatSegmentsNew := make([]string, 0, len(realSegments))
	for _, x := range paths {
		if len(x.SplatSegments) > len(splatSegmentsNew) {
			splatSegmentsNew = x.SplatSegments
		}
	}

	return splatSegmentsNew, paths
}

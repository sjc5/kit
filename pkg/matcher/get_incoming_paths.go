package matcher

import (
	"fmt"

	"github.com/sjc5/kit/pkg/router"
)

func getIncomingPaths(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	return getIncomingPathsNew(registeredPaths, realSegments)
}

func getIncomingPathsNew(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	// Obviously do this setup elsewhere!
	r := router.NewRouter()
	for _, registeredPath := range registeredPaths {
		r.AddRouteWithSegments(router.ParseSegments(registeredPath.Pattern), registeredPath.PathType == PathTypes.Index)
	}

	matches, ok := r.FindAllMatches(realSegments)
	if !ok {
		return nil
	}

	var localMatches []*Match

	for _, match := range matches {
		var rp *RegisteredPath
		for _, registeredPath := range registeredPaths {
			if registeredPath.Pattern == match.Pattern && registeredPath.PathType == PathType(match.LastSegmentType) {
				rp = registeredPath
				break
			}
		}
		if rp == nil {
			fmt.Println("*********** NOT FOUND", "*", match.Pattern, "*", match.SplatSegments, match.Params)
			panic("registered path not found")
		}

		localMatches = append(localMatches, &Match{
			RegisteredPath: rp,
			Results: &Results{
				Params:             match.Params,
				SplatSegments:      match.SplatSegments,
				Score:              0,
				RealSegmentsLength: len(realSegments),
			},
		})
	}

	return localMatches
}

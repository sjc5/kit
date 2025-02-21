package matcher

import "github.com/sjc5/kit/pkg/router"

// Results struct {
// 	Params             Params
// 	SplatSegments      []string
// 	Score              int
// 	RealSegmentsLength int
// }

// RegisteredPath struct {
// 	Pattern  string   `json:"pattern"`
// 	Segments []string `json:"segments"`
// 	PathType PathType `json:"routeType"`
// }

// Match struct {
// 	*RegisteredPath
// 	*Results
// }

func getIncomingPaths(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	// return getIncomingPathsOld(registeredPaths, realSegments)
	return getIncomingPathsNew(registeredPaths, realSegments)
}

func getIncomingPathsNew(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	// Obviously do this setup elsewhere!
	r := router.NewRouter()
	for _, registeredPath := range registeredPaths {
		r.AddRouteWithSegments(registeredPath.Segments)
	}

	matches, ok := r.FindAllMatches(realSegments)

	if !ok {
		return nil
	}

	var localMatches []*Match

	for _, match := range matches {
		// find registered path
		var rp *RegisteredPath

		for _, registeredPath := range registeredPaths {
			if registeredPath.Pattern == match.Pattern {
				rp = registeredPath
				break
			}
		}

		localMatches = append(localMatches, &Match{
			RegisteredPath: rp,
			Results: &Results{
				Params:             match.Params,
				SplatSegments:      match.SplatSegments,
				Score:              match.Score,
				RealSegmentsLength: len(realSegments),
			},
		})
	}

	return localMatches
}

/////// OLD IMPL

func getIncomingPathsOld(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	incomingPaths := make([]*Match, 0, 4)

	for _, registeredPath := range registeredPaths {
		results, ok := matchCore(registeredPath.Segments, realSegments)
		if ok {
			incomingPaths = append(incomingPaths, &Match{
				RegisteredPath: registeredPath,
				Results:        results,
			})
		}
	}

	return incomingPaths
}

func matchCore(patternSegments []string, realSegments []string) (*Results, bool) {
	if len(patternSegments) > 0 {
		if patternSegments[len(patternSegments)-1] == "_index" {
			patternSegments = patternSegments[:len(patternSegments)-1]
		}
	}

	if len(patternSegments) > len(realSegments) {
		return nil, false
	}

	realSegmentsLength := len(realSegments)
	params := make(Params)
	score := 0
	var splatSegments []string

	for i, patternSegment := range patternSegments {
		if i >= len(realSegments) {
			return nil, false
		}

		isLastSegment := i == len(patternSegments)-1

		switch {
		case patternSegment == realSegments[i]:
			score += 3 // Exact match
		case patternSegment == "$":
			score += 1 // Splat segment
			if isLastSegment {
				splatSegments = realSegments[i:]
			}
		case len(patternSegment) > 0 && patternSegment[0] == '$':
			score += 2 // Dynamic parameter
			params[patternSegment[1:]] = realSegments[i]
		default:
			return nil, false
		}
	}

	results := &Results{
		Params:             params,
		Score:              score,
		RealSegmentsLength: realSegmentsLength,
	}

	if splatSegments != nil {
		results.SplatSegments = splatSegments
	}

	return results, true
}

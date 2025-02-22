package matcher

import (
	"fmt"

	"github.com/sjc5/kit/pkg/router"
)

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
			if registeredPath.Pattern == match.Pattern {
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

/////// OLD IMPL

// ****************
// /$ [$] lone-splat
// / [] index
// /articles [articles] index
// /articles/test/articles [articles test articles] index
// /bear/$bear_id/$ [bear $bear_id $] ends-in-splat
// /bear/$bear_id [bear $bear_id] dynamic
// /bear [bear] index
// /bear [bear] static
// /dashboard/$ [dashboard $] ends-in-splat
// /dashboard [dashboard] index
// /dashboard/customers/$customer_id [dashboard customers $customer_id] index
// /dashboard/customers/$customer_id/orders/$order_id [dashboard customers $customer_id orders $order_id] dynamic
// /dashboard/customers/$customer_id/orders [dashboard customers $customer_id orders] index
// /dashboard/customers/$customer_id/orders [dashboard customers $customer_id orders] static
// /dashboard/customers/$customer_id [dashboard customers $customer_id] dynamic
// /dashboard/customers [dashboard customers] index
// /dashboard/customers [dashboard customers] static
// /dashboard [dashboard] static
// /dynamic-index/$pagename [dynamic-index $pagename] index
// /dynamic-index/index [dynamic-index index] static
// /lion/$ [lion $] ends-in-splat
// /lion [lion] index
// /lion [lion] static
// /tiger/$tiger_id/$ [tiger $tiger_id $] ends-in-splat
// /tiger/$tiger_id/$tiger_cub_id [tiger $tiger_id $tiger_cub_id] dynamic
// /tiger/$tiger_id [tiger $tiger_id] index
// /tiger/$tiger_id [tiger $tiger_id] dynamic
// /tiger [tiger] index
// /tiger [tiger] static

func getIncomingPathsOld(registeredPaths RegisteredPaths, realSegments []string) []*Match {
	incomingPaths := make([]*Match, 0, 4)

	for _, registeredPath := range registeredPaths {
		// if i == 0 {
		// 	fmt.Println("****************")
		// }

		// fmt.Println(registeredPath.Pattern, registeredPath.Segments, registeredPath.PathType)

		results, ok := matchCore(router.ParseSegments(registeredPath.Pattern), realSegments)
		if ok {
			// if registeredPath.PathType == PathTypes.Index {
			// 	results.Score += 1
			// }
			incomingPaths = append(incomingPaths, &Match{
				RegisteredPath: registeredPath,
				Results:        results,
			})
		}
	}

	return incomingPaths
}

const (
	scoreStatic  = 3
	scoreDynamic = 2
	scoreSplat   = 1
)

func matchCore(patternSegments []string, realSegments []string) (*Results, bool) {
	if len(patternSegments) == 0 {
		if len(realSegments) > 0 {
			return nil, false
		}
		if len(realSegments) == 0 {
			return &Results{
				Score:              scoreStatic,
				RealSegmentsLength: 0,
			}, true
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
			score += scoreStatic // Exact match
		case patternSegment == "$":
			score += scoreSplat // Splat segment
			if isLastSegment {
				splatSegments = realSegments[i:]
			}
		case len(patternSegment) > 0 && patternSegment[0] == '$':
			score += scoreDynamic // Dynamic parameter
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

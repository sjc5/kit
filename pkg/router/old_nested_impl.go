package router

import (
	"slices"
	"sort"
	"strings"
)

/////////////////////////////////////////////////////////////////////
/////// PUBLIC API
/////////////////////////////////////////////////////////////////////

type (
	PathType string

	Results struct {
		Params             Params
		SplatSegments      []string
		Score              int
		RealSegmentsLength int
	}

	RegisteredRoute_OldNestedImpl struct {
		Pattern  string   `json:"pattern"`
		Segments []string `json:"segments"`
		PathType PathType `json:"routeType"`
	}
)

var (
	PathTypes = struct {
		UltimateCatch    PathType // Lone splat segment
		NonUltimateSplat PathType // Ends in splat segment
		StaticLayout     PathType // Ends in static segment
		DynamicLayout    PathType // Ends in dynamic segment
		Index            PathType // Ends in index segment (only relevant for nested routing)
	}{
		UltimateCatch:    "lone-splat",
		NonUltimateSplat: "splat",
		StaticLayout:     "static",
		DynamicLayout:    "dynamic",
		Index:            "index",
	}
)

/////////////////////////////////////////////////////////////////////
/////// MESSY GET MATCHING PATHS IMPLEMENTATION AND HELPERS
/////////////////////////////////////////////////////////////////////

type groupedBySegmentLength map[int][]*Match

func matchCore(patternSegments []string, realSegments []string) (*Results, bool) {
	if len(patternSegments) > 0 {
		if patternSegments[len(patternSegments)-1] == "_index" || patternSegments[len(patternSegments)-1] == "" {
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

func (router *Router) getMatchingPathsInternal(realPath string) ([]*Match, bool) {
	realSegments := ParseSegments(realPath)
	realSegmentsLength := len(realSegments)
	incomingPaths := make([]*Match, 0, 4)

	for _, rr := range router.StaticRegisteredRoutes {
		segmentVals := make([]string, 0, len(rr.Segments))
		for _, segment := range rr.Segments {
			segmentVals = append(segmentVals, segment.Value)
		}
		results, ok := matchCore(segmentVals, realSegments)
		if ok {
			incomingPaths = append(incomingPaths, &Match{
				RegisteredRoute: rr,
				Params:          results.Params,
				SplatValues:     results.SplatSegments,
				Score:           results.Score,
			})
		}
	}
	for _, rr := range router.DynamicRegisteredRoutes {
		segmentVals := make([]string, 0, len(rr.Segments))
		for _, segment := range rr.Segments {
			segmentVals = append(segmentVals, segment.Value)
		}
		results, ok := matchCore(segmentVals, realSegments)
		if ok {
			incomingPaths = append(incomingPaths, &Match{
				RegisteredRoute: rr,
				Params:          results.Params,
				SplatValues:     results.SplatSegments,
				Score:           results.Score,
			})
		}
	}

	paths := make([]*Match, 0, len(incomingPaths))

	for _, x := range incomingPaths {
		// if it's dash route (home), no need to compare segments length
		if realSegmentsLength == 0 {
			paths = append(paths, x)
			continue
		}

		var indexAdjustedRealSegmentsLength int
		if x.isIndex {
			indexAdjustedRealSegmentsLength = realSegmentsLength + 1
		} else {
			indexAdjustedRealSegmentsLength = realSegmentsLength
		}

		// make sure any remaining matches are not longer than the path itself
		shouldMoveOn := len(x.Segments) <= indexAdjustedRealSegmentsLength
		if !shouldMoveOn {
			continue
		}

		// now we need to remove ineligible indices
		if !x.isIndex {
			// if not an index, then you're already confirmed good
			paths = append(paths, x)
			continue
		}

		truthySegments := make([]string, 0, len(x.Segments))
		for _, segment := range x.Segments {
			if len(segment.Value) > 0 {
				truthySegments = append(truthySegments, segment.Value)
			}
		}
		pathSegments := make([]string, 0, realSegmentsLength)
		for _, segment := range realSegments {
			if len(segment) > 0 {
				pathSegments = append(pathSegments, segment)
			}
		}
		if len(truthySegments) == len(pathSegments) {
			paths = append(paths, x)
		}
	}

	// if there are multiple matches, filter out the ultimate catch-all
	if len(paths) > 1 {
		nonUltimateCatchPaths := make([]*Match, 0, len(paths))
		for _, x := range paths {
			if !x.isUltimateCatch {
				nonUltimateCatchPaths = append(nonUltimateCatchPaths, x)
			}
		}
		paths = nonUltimateCatchPaths
	}

	// if only one match now, return it
	if len(paths) == 1 {
		// if paths[0].isUltimateCatch {
		// 	splatSegments = getBaseSplatSegments(realSegments)
		// }
		return paths, true
	}

	// now we only have real child paths

	// these are essentially any matching static layout routes
	var definiteMatches []*Match // static layout matches
	for _, x := range paths {
		if x.isStaticLayout {
			definiteMatches = append(definiteMatches, x)
		}
	}

	highestScoresBySegmentLengthOfDefiniteMatches := getHighestScoresBySegmentLength(definiteMatches)

	// the "maybe matches" need to compete with each other
	// they also need some more complicated logic

	groupedBySegmentLength := make(groupedBySegmentLength)

	for _, x := range paths {
		if !x.isStaticLayout {
			segmentLength := len(x.Segments)

			highestScoreForThisSegmentLength, exists := highestScoresBySegmentLengthOfDefiniteMatches[segmentLength]

			if !exists || x.Score > highestScoreForThisSegmentLength {
				if groupedBySegmentLength[segmentLength] == nil {
					groupedBySegmentLength[segmentLength] = []*Match{}
				}
				groupedBySegmentLength[segmentLength] = append(groupedBySegmentLength[segmentLength], x)
			}
		}
	}

	sortedGroupedBySegmentLength := getSortedGroupedBySegmentLength(groupedBySegmentLength)

	var xformedMaybes []*Match
	var wildcardSplat *Match = nil
	for _, paths := range sortedGroupedBySegmentLength {
		winner := paths[0]
		highestScore := winner.Score
		var indexCandidate *Match = nil

		for _, path := range paths {
			if path.isIndex && realSegmentsLength < len(path.Segments) {
				if indexCandidate == nil {
					indexCandidate = path
				} else {
					if path.Score > indexCandidate.Score {
						indexCandidate = path
					}
				}
			}
			if path.Score > highestScore {
				highestScore = path.Score
				winner = path
			}
		}

		if indexCandidate != nil {
			winner = indexCandidate
		}

		// find non ultimate splat
		splat := findNonUltimateSplat(paths)

		if splat != nil {
			if wildcardSplat == nil || splat.Score > wildcardSplat.Score {
				wildcardSplat = splat
			}

			// splatSegments = getSplatSegmentsFromWinningPath(winner, realSegments)
		}

		// ok, problem
		// in the situation where we have a dynamic folder name with an index file within,
		// we need to make sure that other static-layout paths win over it
		// that's what this code is for

		winnerIsDynamicIndex := getWinnerIsDynamicIndex(winner)

		definiteMatchesShouldOverride := false
		if winnerIsDynamicIndex {
			for _, x := range definiteMatches {
				a := x.isStaticLayout
				var c bool
				if len(x.Segments) >= 1 && len(winner.Segments) >= 2 {
					lastSegmentOfX := x.Segments[len(x.Segments)-1]
					secondToLastSegmentOfWinner := winner.Segments[len(winner.Segments)-2]
					c = lastSegmentOfX != secondToLastSegmentOfWinner
				}
				d := x.Score > winner.Score
				if a && c && d {
					definiteMatchesShouldOverride = true
					break
				}
			}
		}

		if !definiteMatchesShouldOverride {
			xformedMaybes = append(xformedMaybes, winner)
		}
	}

	maybeFinalPaths := getMaybeFinalPaths(definiteMatches, xformedMaybes)

	if len(maybeFinalPaths) > 0 {
		lastPath := maybeFinalPaths[len(maybeFinalPaths)-1]

		// get index-adjusted segments length
		var lastPathSegmentsLengthConstructive int
		if lastPath.isIndex {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments) - 1
		} else {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments)
		}

		splatIsTooFarOut := lastPathSegmentsLengthConstructive > realSegmentsLength
		splatIsNeeded := lastPathSegmentsLengthConstructive < realSegmentsLength
		isNotASplat := !lastPath.isNonUltimateSplat
		weNeedADifferentSplat := splatIsTooFarOut || (splatIsNeeded && isNotASplat)

		if weNeedADifferentSplat {
			if wildcardSplat != nil {
				maybeFinalPaths[len(maybeFinalPaths)-1] = wildcardSplat
				// splatSegments = getSplatSegmentsFromWinningPath(wildcardSplat, realSegments)
			} else {
				// splatSegments = getBaseSplatSegments(realSegments)
				var filteredPaths []*Match
				for _, x := range incomingPaths {
					if x.isUltimateCatch {
						filteredPaths = append(filteredPaths, x)
						break
					}
				}
				return filteredPaths, len(filteredPaths) > 0
			}
		}
	}

	// if a dynamic layout is adjacent and before an index, we need to remove it
	// IF the index does not share the same dynamic segment
	for i := 0; i < len(maybeFinalPaths); i++ {
		current := maybeFinalPaths[i]

		if i+1 < len(maybeFinalPaths) {
			next := *maybeFinalPaths[i+1]

			if current.isDynamicLayout && next.isIndex {
				currentDynamicSegment := current.Segments[len(current.Segments)-1].Value
				nextDynamicSegment := next.Segments[len(next.Segments)-2].Value
				if currentDynamicSegment != nextDynamicSegment {
					maybeFinalPaths = append(maybeFinalPaths[:i], maybeFinalPaths[i+1:]...)
				}
			}
		}
	}

	return maybeFinalPaths, len(maybeFinalPaths) > 0
}

func findNonUltimateSplat(paths []*Match) *Match {
	for _, path := range paths {
		if path.isNonUltimateSplat {
			return path // Return a pointer to the matching path
		}
	}
	return nil // Return nil if no matching path is found
}

func getSortedGroupedBySegmentLength(groupedBySegmentLength groupedBySegmentLength) [][]*Match {
	keys := make([]int, 0, len(groupedBySegmentLength))
	for k := range groupedBySegmentLength {
		keys = append(keys, k)
	}

	// Sort the keys in ascending order
	sort.Ints(keys)

	sortedGroupedBySegmentLength := make([][]*Match, 0, len(groupedBySegmentLength))
	for _, k := range keys {
		sortedGroupedBySegmentLength = append(sortedGroupedBySegmentLength, groupedBySegmentLength[k])
	}

	return sortedGroupedBySegmentLength
}

func getHighestScoresBySegmentLength(matches []*Match) map[int]int {
	highestScores := make(map[int]int, len(matches))

	for _, match := range matches {
		segmentLength := len(match.Segments)
		if currentScore, exists := highestScores[segmentLength]; !exists || match.Score > currentScore {
			highestScores[segmentLength] = match.Score
		}
	}

	return highestScores
}

func getWinnerIsDynamicIndex(winner *Match) bool {
	segmentsLen := len(winner.Segments)
	if winner.isIndex && segmentsLen >= 2 {
		return winner.Segments[segmentsLen-2].Type == SegmentTypes.Dynamic
	}
	return false
}

func getMaybeFinalPaths(definiteMatches, xformedMaybes []*Match) []*Match {
	maybeFinalPaths := append(definiteMatches, xformedMaybes...)
	slices.SortStableFunc(maybeFinalPaths, func(i, j *Match) int {
		return len(i.Segments) - len(j.Segments)
	})
	return maybeFinalPaths
}

/////////////////////////////////////////////////////////////////////
/////// BUILD TIME
/////////////////////////////////////////////////////////////////////

func PatternToRegisteredPath(pattern string) *RegisteredRoute_OldNestedImpl {
	rawSegments := ParseSegments(pattern)
	rawSegmentsLen := len(rawSegments)

	segments := make([]string, 0, rawSegmentsLen)
	for _, segment := range rawSegments {
		// Skip double underscore segments
		if len(segment) > 1 && segment[0] == '_' && segment[1] == '_' {
			continue
		}

		// Convert _index to empty string
		if segment == "_index" {
			segments = append(segments, "")
		} else {
			segments = append(segments, segment)
		}
	}

	if len(segments) == 0 {
		return &RegisteredRoute_OldNestedImpl{Pattern: "/", Segments: []string{""}, PathType: PathTypes.Index}
	}

	lastSegment := segments[len(segments)-1]

	var routeType PathType

	switch {
	case lastSegment == "":
		routeType = PathTypes.Index
	case rawSegmentsLen == 1 && lastSegment == "$":
		routeType = PathTypes.UltimateCatch
	case lastSegment == "$":
		routeType = PathTypes.NonUltimateSplat
	case len(lastSegment) > 1 && lastSegment[0] == '$':
		routeType = PathTypes.DynamicLayout
	default:
		routeType = PathTypes.StaticLayout
	}

	return &RegisteredRoute_OldNestedImpl{
		Pattern:  buildNormalizedPattern(segments, routeType == PathTypes.Index),
		Segments: segments,
		PathType: routeType,
	}
}

// Helper function to build normalized pattern
func buildNormalizedPattern(segments []string, isIndex bool) string {
	// Filter out empty segments for the pattern
	truthySegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			truthySegments = append(truthySegments, segment)
		}
	}

	// Build the pattern
	pattern := "/" + strings.Join(truthySegments, "/")
	if hasTrailingSlashButIsNotRoot(pattern) {
		pattern = pattern[:len(pattern)-1]
	}

	// Add _index if needed
	if isIndex {
		if pattern == "/" {
			pattern += "_index"
		} else {
			pattern += "/_index"
		}
	}

	return pattern
}

func hasTrailingSlashButIsNotRoot(path string) bool {
	return path != "/" && len(path) > 1 && path[len(path)-1] == '/'
}

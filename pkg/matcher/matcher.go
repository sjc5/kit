package matcher

import (
	"slices"
	"sort"

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
		Pattern  string   `json:"pattern"`
		Segments []string `json:"segments"`
		PathType PathType `json:"routeType"`
	}

	Match struct {
		*RegisteredPath
		*Results
	}
)

var (
	GetMatchingPaths = getMatchingPathsInternal

	PathTypes = struct {
		UltimateCatch    PathType // Lone splat segment
		NonUltimateSplat PathType // Ends in splat segment
		StaticLayout     PathType // Ends in static segment
		DynamicLayout    PathType // Ends in dynamic segment
		Index            PathType // Ends in index segment (only relevant for nested routing)
	}{
		UltimateCatch:    "lone-splat",
		NonUltimateSplat: "ends-in-splat",
		StaticLayout:     "static",
		DynamicLayout:    "dynamic",
		Index:            "index",
	}
)

/////////////////////////////////////////////////////////////////////
/////// MESSY GET MATCHING PATHS IMPLEMENTATION AND HELPERS
/////////////////////////////////////////////////////////////////////

type groupedBySegmentLength map[int][]*Match

func getMatchingPathsInternal(registeredPaths RegisteredPaths, realPath string) ([]string, []*Match) {
	realSegments := router.ParseSegments(realPath)
	incomingPaths := getIncomingPaths(registeredPaths, realSegments)

	paths := make([]*Match, 0, len(incomingPaths))

	for _, x := range incomingPaths {
		// if it's dash route (home), no need to compare segments length
		if x.RealSegmentsLength == 0 {
			paths = append(paths, x)
			continue
		}

		var indexAdjustedRealSegmentsLength int
		if x.PathType == PathTypes.Index {
			indexAdjustedRealSegmentsLength = x.RealSegmentsLength + 1
		} else {
			indexAdjustedRealSegmentsLength = x.RealSegmentsLength
		}

		// make sure any remaining matches are not longer than the path itself
		shouldMoveOn := len(x.Segments) <= indexAdjustedRealSegmentsLength
		if !shouldMoveOn {
			continue
		}

		// now we need to remove ineligible indices
		if x.PathType != PathTypes.Index {
			// if not an index, then you're already confirmed good
			paths = append(paths, x)
			continue
		}

		nonIndexSegments := make([]string, 0, len(x.Segments))
		for _, segment := range x.Segments {
			if segment != "_index" {
				nonIndexSegments = append(nonIndexSegments, segment)
			}
		}
		pathSegments := make([]string, 0, x.RealSegmentsLength)
		for _, segment := range realSegments {
			if segment != "_index" {
				pathSegments = append(pathSegments, segment)
			}
		}
		if len(nonIndexSegments) == len(pathSegments) {
			paths = append(paths, x)
		}
	}

	// if there are multiple matches, filter out the ultimate catch-all
	if len(paths) > 1 {
		nonUltimateCatchPaths := make([]*Match, 0, len(paths))
		for _, x := range paths {
			if x.PathType != PathTypes.UltimateCatch {
				nonUltimateCatchPaths = append(nonUltimateCatchPaths, x)
			}
		}
		paths = nonUltimateCatchPaths
	}

	var splatSegments []string

	// if only one match now, return it
	if len(paths) == 1 {
		if paths[0].PathType == PathTypes.UltimateCatch {
			splatSegments = getBaseSplatSegments(realSegments)
		}
		return splatSegments, paths
	}

	// now we only have real child paths

	// these are essentially any matching static layout routes
	var definiteMatches []*Match // static layout matches
	for _, x := range paths {
		if x.PathType == PathTypes.StaticLayout {
			definiteMatches = append(definiteMatches, x)
		}
	}

	highestScoresBySegmentLengthOfDefiniteMatches := getHighestScoresBySegmentLength(definiteMatches)

	// the "maybe matches" need to compete with each other
	// they also need some more complicated logic

	groupedBySegmentLength := make(groupedBySegmentLength)

	for _, x := range paths {
		if x.PathType != PathTypes.StaticLayout {
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
			if path.PathType == PathTypes.Index && path.RealSegmentsLength < len(path.Segments) {
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

			splatSegments = getSplatSegmentsFromWinningPath(winner, realSegments)
		}

		// ok, problem
		// in the situation where we have a dynamic folder name with an index file within,
		// we need to make sure that other static-layout paths win over it
		// that's what this code is for

		winnerIsDynamicIndex := getWinnerIsDynamicIndex(winner)

		definiteMatchesShouldOverride := false
		if winnerIsDynamicIndex {
			for _, x := range definiteMatches {
				a := x.PathType == PathTypes.StaticLayout
				b := x.RealSegmentsLength == winner.RealSegmentsLength
				var c bool
				if len(x.Segments) >= 1 && len(winner.Segments) >= 2 {
					lastSegmentOfX := x.Segments[len(x.Segments)-1]
					secondToLastSegmentOfWinner := winner.Segments[len(winner.Segments)-2]
					c = lastSegmentOfX != secondToLastSegmentOfWinner
				}
				d := x.Score > winner.Score
				if a && b && c && d {
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
		if lastPath.PathType == PathTypes.Index {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments) - 1
		} else {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments)
		}

		splatIsTooFarOut := lastPathSegmentsLengthConstructive > lastPath.RealSegmentsLength
		splatIsNeeded := lastPathSegmentsLengthConstructive < lastPath.RealSegmentsLength
		isNotASplat := lastPath.PathType != PathTypes.NonUltimateSplat
		weNeedADifferentSplat := splatIsTooFarOut || (splatIsNeeded && isNotASplat)

		if weNeedADifferentSplat {
			if wildcardSplat != nil {
				maybeFinalPaths[len(maybeFinalPaths)-1] = wildcardSplat
				splatSegments = getSplatSegmentsFromWinningPath(wildcardSplat, realSegments)
			} else {
				splatSegments = getBaseSplatSegments(realSegments)
				var filteredPaths []*Match
				for _, x := range incomingPaths {
					if x.PathType == PathTypes.UltimateCatch {
						filteredPaths = append(filteredPaths, x)
						break
					}
				}
				return splatSegments, filteredPaths
			}
		}
	}

	// if a dynamic layout is adjacent and before an index, we need to remove it
	// IF the index does not share the same dynamic segment
	for i := 0; i < len(maybeFinalPaths); i++ {
		current := maybeFinalPaths[i]

		if i+1 < len(maybeFinalPaths) {
			next := *maybeFinalPaths[i+1]

			if current.PathType == PathTypes.DynamicLayout && next.PathType == PathTypes.Index {
				currentDynamicSegment := current.Segments[len(current.Segments)-1]
				nextDynamicSegment := next.Segments[len(next.Segments)-2]
				if currentDynamicSegment != nextDynamicSegment {
					maybeFinalPaths = append(maybeFinalPaths[:i], maybeFinalPaths[i+1:]...)
				}
			}
		}
	}

	return splatSegments, maybeFinalPaths
}

func findNonUltimateSplat(paths []*Match) *Match {
	for _, path := range paths {
		if path.PathType == PathTypes.NonUltimateSplat {
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

func getSplatSegmentsFromWinningPath(winner *Match, realSegments []string) []string {
	filteredData := make([]string, 0, len(realSegments))
	for _, segment := range realSegments {
		if segment != "_index" {
			filteredData = append(filteredData, segment)
		}
	}

	numOfNonSplatSegments := 0
	for _, x := range winner.Segments {
		if x != "$" {
			numOfNonSplatSegments++
		}
	}

	numOfSplatSegments := len(filteredData) - numOfNonSplatSegments
	if numOfSplatSegments > 0 {
		final := filteredData[len(filteredData)-numOfSplatSegments:]
		return final
	} else {
		return []string{}
	}
}

func getWinnerIsDynamicIndex(winner *Match) bool {
	segmentsLen := len(winner.Segments)
	if winner.PathType == PathTypes.Index && segmentsLen >= 2 {
		return isDynamicSegment(winner.Segments[segmentsLen-2])
	}
	return false
}

func isDynamicSegment(segment string) bool {
	return len(segment) > 1 && segment[0] == '$'
}

func getMaybeFinalPaths(definiteMatches, xformedMaybes []*Match) []*Match {
	maybeFinalPaths := append(definiteMatches, xformedMaybes...)
	slices.SortStableFunc(maybeFinalPaths, func(i, j *Match) int {
		return len(i.Segments) - len(j.Segments)
	})
	return maybeFinalPaths
}

func getBaseSplatSegments(realSegments []string) []string {
	var splatSegments []string
	for _, segment := range realSegments {
		if segment != "_index" {
			splatSegments = append(splatSegments, segment)
		}
	}
	return splatSegments
}

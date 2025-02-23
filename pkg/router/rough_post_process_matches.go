package router

import (
	"slices"
	"sort"
)

type groupedBySegmentLength map[int][]*Match

func (router *RouterBest) getMatchingPathsInternal(matches []*Match, realSegments []string) ([]string, []*Match) {
	realSegmentsLen := len(realSegments)
	paths := make([]*Match, 0, len(matches))

	for _, x := range matches {
		// if it's dash route (home), no need to compare segments length
		if realSegmentsLen == 0 {
			paths = append(paths, x)
			continue
		}

		var indexAdjustedRealSegmentsLength int
		if x.LastSegmentIsIndex() {
			indexAdjustedRealSegmentsLength = realSegmentsLen + 1
		} else {
			indexAdjustedRealSegmentsLength = realSegmentsLen
		}

		// make sure any remaining matches are not longer than the path itself
		shouldMoveOn := len(x.Segments) <= indexAdjustedRealSegmentsLength
		if !shouldMoveOn {
			continue
		}

		// now we need to remove ineligible indices
		if !x.LastSegmentIsIndex() {
			// if not an index, then you're already confirmed good
			paths = append(paths, x)
			continue
		}

		nonIndexSegments := make([]string, 0, len(x.Segments))
		for _, segment := range x.Segments {
			if segment.Value != router.NestedIndexSignifier {
				nonIndexSegments = append(nonIndexSegments, segment.Value)
			}
		}
		pathSegments := make([]string, 0, realSegmentsLen)
		for _, segment := range realSegments {
			if segment != router.NestedIndexSignifier {
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
			if !x.LastSegmentIsUltimateCatch() {
				nonUltimateCatchPaths = append(nonUltimateCatchPaths, x)
			}
		}
		paths = nonUltimateCatchPaths
	}

	var splatSegments []string

	// if only one match now, return it
	if len(paths) == 1 {
		if paths[0].LastSegmentIsUltimateCatch() {
			splatSegments = router.getBaseSplatSegments(realSegments)
		}
		return splatSegments, paths
	}

	// now we only have real child paths

	// these are essentially any matching static layout routes
	var definiteMatches []*Match // static layout matches
	for _, x := range paths {
		if x.LastSegmentIsStaticLayout() {
			definiteMatches = append(definiteMatches, x)
		}
	}

	highestScoresBySegmentLengthOfDefiniteMatches := getHighestScoresBySegmentLength(definiteMatches)

	// the "maybe matches" need to compete with each other
	// they also need some more complicated logic

	groupedBySegmentLength := make(groupedBySegmentLength)

	for _, x := range paths {
		if !x.LastSegmentIsStaticLayout() {
			segmentLength := len(x.Segments)

			highestScoreForThisSegmentLength, exists := highestScoresBySegmentLengthOfDefiniteMatches[segmentLength]

			if !exists || x.Score > highestScoreForThisSegmentLength {
				if groupedBySegmentLength[segmentLength] == nil {
					groupedBySegmentLength[segmentLength] = make([]*Match, 0, 1)
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
			if path.LastSegmentIsIndex() && realSegmentsLen < len(path.Segments) {
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

			splatSegments = router.getSplatSegmentsFromWinningPath(winner, realSegments)
		}

		// ok, problem
		// in the situation where we have a dynamic folder name with an index file within,
		// we need to make sure that other static-layout paths win over it
		// that's what this code is for

		winnerIsDynamicIndex := winner.LastSegmentIsIndexPrecededByDynamic()

		definiteMatchesShouldOverride := false
		if winnerIsDynamicIndex {
			for _, x := range definiteMatches {
				a := x.LastSegmentIsStaticLayout()
				b := realSegmentsLen == realSegmentsLen
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
		if lastPath.LastSegmentIsIndex() {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments) - 1
		} else {
			lastPathSegmentsLengthConstructive = len(lastPath.Segments)
		}

		splatIsTooFarOut := lastPathSegmentsLengthConstructive > realSegmentsLen
		splatIsNeeded := lastPathSegmentsLengthConstructive < realSegmentsLen
		isNotASplat := !lastPath.LastSegmentIsNonUltimateSplat()
		weNeedADifferentSplat := splatIsTooFarOut || (splatIsNeeded && isNotASplat)

		if weNeedADifferentSplat {
			if wildcardSplat != nil {
				maybeFinalPaths[len(maybeFinalPaths)-1] = wildcardSplat
				splatSegments = router.getSplatSegmentsFromWinningPath(wildcardSplat, realSegments)
			} else {
				splatSegments = router.getBaseSplatSegments(realSegments)
				var filteredPaths []*Match
				for _, x := range matches { // not sure why we are looping on initial matches now
					if x.LastSegmentIsUltimateCatch() {
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

			if current.LastSegmentIsDynamicLayout() && next.LastSegmentIsIndex() {
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
		if path.LastSegmentIsNonUltimateSplat() {
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

func (router *RouterBest) getSplatSegmentsFromWinningPath(winner *Match, realSegments []string) []string {
	filteredData := make([]string, 0, len(realSegments))
	for _, segment := range realSegments {
		if segment != router.NestedIndexSignifier {
			filteredData = append(filteredData, segment)
		}
	}

	numOfNonSplatSegments := 0
	for _, x := range winner.Segments {
		if x.Value != "$" {
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

func (router *RouterBest) getBaseSplatSegments(realSegments []string) []string {
	var splatSegments []string
	for _, segment := range realSegments {
		if segment != router.NestedIndexSignifier {
			splatSegments = append(splatSegments, segment)
		}
	}
	return splatSegments
}

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
	paths := getIncomingPaths(registeredPaths, realSegments)

	splatSegmentsNew := make([]string, 0, len(realSegments))
	for _, x := range paths {
		if len(x.SplatSegments) > len(splatSegmentsNew) {
			splatSegmentsNew = x.SplatSegments
		}
	}

	return splatSegmentsNew, paths

	// // if there are multiple matches, filter out the ultimate catch-all
	// if len(paths) > 1 {
	// 	nonUltimateCatchPaths := make([]*Match, 0, len(paths))
	// 	for _, x := range paths {
	// 		if x.PathType != PathTypes.UltimateCatch {
	// 			nonUltimateCatchPaths = append(nonUltimateCatchPaths, x)
	// 		}
	// 	}
	// 	paths = nonUltimateCatchPaths
	// }

	// var splatSegments []string

	// // if only one match now, return it
	// if len(paths) == 1 {
	// 	if paths[0].PathType == PathTypes.UltimateCatch {
	// 		splatSegments = getBaseSplatSegments(realSegments)
	// 	}
	// 	return splatSegments, paths
	// }

	// // now we only have real child paths

	// // these are essentially any matching static layout routes
	// var definiteMatches []*Match // static layout matches
	// for _, x := range paths {
	// 	if x.PathType == PathTypes.StaticLayout {
	// 		definiteMatches = append(definiteMatches, x)
	// 	}
	// }

	// highestScoresBySegmentLengthOfDefiniteMatches := getHighestScoresBySegmentLength(definiteMatches)

	// // the "maybe matches" need to compete with each other
	// // they also need some more complicated logic

	// groupedBySegmentLength := make(groupedBySegmentLength)

	// for _, x := range paths {
	// 	if x.PathType != PathTypes.StaticLayout {
	// 		segmentLength := len(x.Segments)

	// 		highestScoreForThisSegmentLength, exists := highestScoresBySegmentLengthOfDefiniteMatches[segmentLength]

	// 		if !exists || x.Score > highestScoreForThisSegmentLength {
	// 			if groupedBySegmentLength[segmentLength] == nil {
	// 				groupedBySegmentLength[segmentLength] = []*Match{}
	// 			}
	// 			groupedBySegmentLength[segmentLength] = append(groupedBySegmentLength[segmentLength], x)
	// 		}
	// 	}
	// }

	// sortedGroupedBySegmentLength := getSortedGroupedBySegmentLength(groupedBySegmentLength)

	// // if x.PathType == PathTypes.Index {
	// // 	// if index is eligible (same length as real segments), it should be added to definite matches
	// // 	definiteMatches = append(definiteMatches, x)
	// // }

	// var xformedMaybes []*Match
	// var wildcardSplat *Match = nil
	// for _, paths := range sortedGroupedBySegmentLength {
	// 	winner := paths[0]
	// 	highestScore := winner.Score
	// 	var indexCandidate *Match = nil

	// 	for _, path := range paths {
	// 		if path.PathType == PathTypes.Index && path.RealSegmentsLength < len(path.Segments) {
	// 			if indexCandidate == nil {
	// 				indexCandidate = path
	// 			} else {
	// 				if path.Score > indexCandidate.Score {
	// 					indexCandidate = path
	// 				}
	// 			}
	// 		}
	// 		if path.Score > highestScore {
	// 			highestScore = path.Score
	// 			winner = path
	// 		}
	// 	}

	// 	if indexCandidate != nil {
	// 		winner = indexCandidate
	// 	}

	// 	// find non ultimate splat
	// 	splat := findNonUltimateSplat(paths)

	// 	if splat != nil {
	// 		if wildcardSplat == nil || splat.Score > wildcardSplat.Score {
	// 			wildcardSplat = splat
	// 		}

	// 		splatSegments = getSplatSegmentsFromWinningPath(winner, realSegments)
	// 	}

	// 	// ok, problem
	// 	// in the situation where we have a dynamic folder name with an index file within,
	// 	// we need to make sure that other static-layout paths win over it
	// 	// that's what this code is for

	// 	winnerIsDynamicIndex := getWinnerIsDynamicIndex(winner)

	// 	definiteMatchesShouldOverride := false
	// 	if winnerIsDynamicIndex {
	// 		for _, x := range definiteMatches {
	// 			a := x.PathType == PathTypes.StaticLayout
	// 			b := x.RealSegmentsLength == winner.RealSegmentsLength
	// 			var c bool
	// 			if len(x.Segments) >= 1 && len(winner.Segments) >= 2 {
	// 				lastSegmentOfX := x.Segments[len(x.Segments)-1]
	// 				secondToLastSegmentOfWinner := winner.Segments[len(winner.Segments)-2]
	// 				c = lastSegmentOfX != secondToLastSegmentOfWinner
	// 			}
	// 			d := x.Score > winner.Score
	// 			if a && b && c && d {
	// 				definiteMatchesShouldOverride = true
	// 				break
	// 			}
	// 		}
	// 	}

	// 	if !definiteMatchesShouldOverride {
	// 		xformedMaybes = append(xformedMaybes, winner)
	// 	}
	// }

	// maybeFinalPaths := getMaybeFinalPaths(definiteMatches, xformedMaybes)

	// if len(maybeFinalPaths) > 0 {
	// 	lastPath := maybeFinalPaths[len(maybeFinalPaths)-1]

	// 	// get index-adjusted segments length
	// 	// var lastPathSegmentsLengthConstructive int
	// 	// if lastPath.PathType == PathTypes.Index {
	// 	// 	lastPathSegmentsLengthConstructive = len(lastPath.Segments) - 1
	// 	// } else {
	// 	// 	lastPathSegmentsLengthConstructive = len(lastPath.Segments)
	// 	// }

	// 	splatIsTooFarOut := len(lastPath.Segments) > lastPath.RealSegmentsLength
	// 	splatIsNeeded := len(lastPath.Segments) < lastPath.RealSegmentsLength
	// 	isNotASplat := lastPath.PathType != PathTypes.NonUltimateSplat
	// 	weNeedADifferentSplat := splatIsTooFarOut || (splatIsNeeded && isNotASplat)

	// 	if realPath == "/lion/123" {
	// 		fmt.Println("maybeFinalPaths")
	// 		for _, x := range maybeFinalPaths {
	// 			fmt.Println(x.Segments, x.PathType, x.RealSegmentsLength)
	// 		}

	// 		fmt.Println("SPLAT IS TOO FAR OUT", splatIsTooFarOut)
	// 		fmt.Println("SPLAT IS NEEDED", splatIsNeeded)
	// 		fmt.Println("IS NOT A SPLAT", isNotASplat)
	// 		fmt.Println("WE NEED A DIFFERENT SPLAT", weNeedADifferentSplat)
	// 	}

	// 	if weNeedADifferentSplat {
	// 		if wildcardSplat != nil {
	// 			maybeFinalPaths[len(maybeFinalPaths)-1] = wildcardSplat
	// 			splatSegments = getSplatSegmentsFromWinningPath(wildcardSplat, realSegments)
	// 		} else {
	// 			splatSegments = getBaseSplatSegments(realSegments)
	// 			var filteredPaths []*Match
	// 			for _, x := range paths {
	// 				if x.PathType == PathTypes.UltimateCatch {
	// 					filteredPaths = append(filteredPaths, x)
	// 					break
	// 				}
	// 			}
	// 			return splatSegments, veryVeryBadFixLaterHandleStripIndexIfNeeded(filteredPaths)
	// 		}
	// 	}
	// }

	// // if a dynamic layout is adjacent and before an index, we need to remove it
	// // IF the index does not share the same dynamic segment
	// for i := 0; i < len(maybeFinalPaths); i++ {
	// 	current := maybeFinalPaths[i]

	// 	if i+1 < len(maybeFinalPaths) {
	// 		next := *maybeFinalPaths[i+1]

	// 		if current.PathType == PathTypes.DynamicLayout && next.PathType == PathTypes.Index {
	// 			currentDynamicSegment := current.Segments[len(current.Segments)-1]
	// 			if len(next.Segments) >= 2 {
	// 				nextDynamicSegment := next.Segments[len(next.Segments)-2]
	// 				if currentDynamicSegment != nextDynamicSegment {
	// 					maybeFinalPaths = append(maybeFinalPaths[:i], maybeFinalPaths[i+1:]...)
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// return splatSegments, veryVeryBadFixLaterHandleStripIndexIfNeeded(maybeFinalPaths)
}

func veryVeryBadFixLaterHandleStripIndexIfNeeded(paths []*Match) []*Match {
	// if paths contains an index and another path with a score higher than such index, remove the index

	// find the index
	var index *Match
	for _, x := range paths {
		if x.PathType == PathTypes.Index {
			index = x
			break
		}
	}

	if index == nil {
		return paths
	}

	// find the highest score
	highestScore := 0
	for _, x := range paths {
		if x.Score > highestScore {
			highestScore = x.Score
		}
	}

	// if the index has a lower score than the highest score, remove it
	if index.Score < highestScore {
		var newPaths []*Match
		for _, x := range paths {
			if x != index {
				newPaths = append(newPaths, x)
			}
		}
		return newPaths
	}

	return paths
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
	filteredData = append(filteredData, realSegments...)

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
		// sort by segment length, and any index paths identical to its comparable should be moved down, and any splat segments should be absolutely last
		if i.RealSegmentsLength == j.RealSegmentsLength {
			iIsIndex := i.PathType == PathTypes.Index
			jIsIndex := j.PathType == PathTypes.Index
			iIsSplat := i.PathType == PathTypes.UltimateCatch || i.PathType == PathTypes.NonUltimateSplat
			jIsSplat := j.PathType == PathTypes.UltimateCatch || j.PathType == PathTypes.NonUltimateSplat

			if iIsIndex && !jIsSplat {
				return 1
			}
			if jIsIndex && !iIsSplat {
				return -1
			}

			return 0
		}
		return i.RealSegmentsLength - j.RealSegmentsLength
	})
	return maybeFinalPaths
}

func getBaseSplatSegments(realSegments []string) []string {
	var splatSegments []string
	splatSegments = append(splatSegments, realSegments...)
	return splatSegments
}

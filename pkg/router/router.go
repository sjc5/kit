package router

import (
	"fmt"
	"slices"
)

// Note -- should we validate that there are no two competing dynamic segments in otherwise matching routes?

// todo -- double check slash handling
func (router *RouterBest) FindBestMatch(realPath string) (*Match, bool) {
	// fast path if totally static
	if rr, ok := router.StaticRegisteredRoutes[realPath]; ok {
		return &Match{RegisteredRoute: rr}, true
	}

	var bestMatch *Match
	for pattern := range router.DynamicRegisteredRoutes {
		if match, ok := router.SimpleMatch(pattern, realPath, false, false); ok {
			if bestMatch == nil || match.Score > bestMatch.Score {
				bestMatch = match
			}
		}
	}

	return bestMatch, bestMatch != nil

	// realSegments := ParseSegments(realPath)
	// traverse, getMatches := router.makeTraverseFunc(realSegments, false)
	// traverse(router.trie.root, 0, 0)
	// matches := getMatches()
	// if len(matches) == 0 {
	// 	return nil, false
	// }
	// bestMatch := matches[0]
	// return bestMatch, bestMatch != nil
}

type MatchesMap = map[string]*Match

// todo -- double check slash handling
func (router *RouterBest) FindAllMatches(realPath string) ([]*Match, bool) {
	realSegments := ParseSegments(realPath)
	matches := make(MatchesMap)

	if len(realSegments) == 0 {
		if rr, ok := router.StaticRegisteredRoutes["/"]; ok {
			matches[rr.Pattern] = &Match{RegisteredRoute: rr, notes: "added via static 1"}
		}
		if rr, ok := router.StaticRegisteredRoutes["/"+router.NestedIndexSignifier]; ok {
			matches[rr.Pattern] = &Match{RegisteredRoute: rr, notes: "added via static 2"}
		}

		return flattenMatches(matches)
	}

	var path string
	var foundFullStatic bool
	for i := 0; i < len(realSegments); i++ {
		path += "/" + realSegments[i]
		if rr, ok := router.StaticRegisteredRoutes[path]; ok {
			matches[rr.Pattern] = &Match{RegisteredRoute: rr, notes: fmt.Sprintf("added via static 3, path: %s", path)}
			if i == len(realSegments)-1 {
				foundFullStatic = true
			}
		}
		if i == len(realSegments)-1 {
			if rr, ok := router.StaticRegisteredRoutes[path+"/"+router.NestedIndexSignifier]; ok {
				matches[rr.Pattern] = &Match{RegisteredRoute: rr, notes: fmt.Sprintf("added via static 4, path: %s", path)}
			}
		}
	}

	if !foundFullStatic {
		for pattern := range router.DynamicRegisteredRoutes {
			if match, ok := router.SimpleMatch(pattern, realPath, true, false); ok {
				match.notes = fmt.Sprintf("added via dynamic, pattern: %s", pattern)
				matches[pattern] = match
			}
			if match, ok := router.SimpleMatch(pattern, realPath, true, true); ok {
				match.notes = fmt.Sprintf("added via dynamic 2, pattern: %s", pattern)
				matches[pattern] = match
			}
		}
	}

	// if there are multiple matches and a catch-all, remove the catch-all
	if _, ok := matches["/$"]; ok {
		if len(matches) > 1 {
			delete(matches, "/$")
		}
	}

	if len(matches) < 2 {
		return flattenMatches(matches)
	}

	var longestSegmentLen int
	longestSegmentMatches := make(MatchesMap)
	for _, match := range matches {
		if len(match.Segments) > longestSegmentLen {
			longestSegmentLen = len(match.Segments)
		}
	}
	for _, match := range matches {
		if len(match.Segments) == longestSegmentLen {
			longestSegmentMatches[match.GetLastSegmentType()] = match
		}
	}

	// if there is any splat or index with a segment length shorter than longest segment length, remove it
	for pattern, match := range matches {
		if len(match.Segments) < longestSegmentLen {
			if match.LastSegmentIsNonUltimateSplat() || match.LastSegmentIsIndex() {
				delete(matches, pattern)
			}
		}
	}

	if len(matches) < 2 {
		return flattenMatches(matches)
	}

	// if the longest segment length items are (1) dynamic, (2) splat, or (3) index, remove them as follows:
	// - if the realSegmentLen equals the longest segment length, prioritize dynamic, then splat, and always remove index
	// - if the realSegmentLen is greater than the longest segment length, prioritize splat, and always remove dynamic and index
	if len(longestSegmentMatches) > 1 {
		if match, indexExists := longestSegmentMatches[SegmentTypes.Index]; indexExists {
			delete(matches, match.Pattern)
		}

		_, dynamicExists := longestSegmentMatches[SegmentTypes.Dynamic]
		_, splatExists := longestSegmentMatches[SegmentTypes.Splat]

		if len(realSegments) == longestSegmentLen && dynamicExists && splatExists {
			delete(matches, longestSegmentMatches[SegmentTypes.Splat].Pattern)
		}
		if len(realSegments) > longestSegmentLen && splatExists && dynamicExists {
			delete(matches, longestSegmentMatches[SegmentTypes.Dynamic].Pattern)
		}
	}

	return flattenMatches(matches)

	// realSegments := ParseSegments(realPath)
	// traverse, getMatches := router.makeTraverseFunc(realSegments, true)
	// traverse(router.trie.root, 0, 0)
	// matches := getMatches()
	// if len(matches) == 0 {
	// 	return nil, false
	// }
	// return matches, true
}

func flattenMatches(matches MatchesMap) ([]*Match, bool) {
	var results []*Match
	for _, match := range matches {
		results = append(results, match)
	}

	slices.SortStableFunc(results, func(i, j *Match) int {
		// if any match is an index, it should be last
		if i.LastSegmentIsIndex() {
			return 1
		}
		if j.LastSegmentIsIndex() {
			return -1
		}

		// else sort by segment length
		return len(i.Segments) - len(j.Segments)
	})

	return results, len(results) > 0
}

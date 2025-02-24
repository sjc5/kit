package router

import (
	"slices"
	"strings"
)

func (m *Matcher) FindNestedMatches(realPath string) ([]*Match, bool) {
	realSegments := ParseSegments(realPath)
	matches := make(matchesMap)

	// Handle empty path case
	if len(realSegments) == 0 {
		if rr, ok := m.staticPatterns["/"]; ok {
			matches[rr.pattern] = &Match{RegisteredPattern: rr}
		}
		if rr, ok := m.staticPatterns[m.slashNestedIndexSignifier]; ok {
			matches[rr.pattern] = &Match{RegisteredPattern: rr}
		}
		return flattenAndSortMatches(matches)
	}

	var pb strings.Builder
	var foundFullStatic bool
	for i := 0; i < len(realSegments); i++ {
		pb.WriteString("/")
		pb.WriteString(realSegments[i])
		if rr, ok := m.staticPatterns[pb.String()]; ok {
			matches[rr.pattern] = &Match{RegisteredPattern: rr}
			if i == len(realSegments)-1 {
				foundFullStatic = true
			}
		}
		if i == len(realSegments)-1 {
			pb.WriteString(m.slashNestedIndexSignifier)
			if rr, ok := m.staticPatterns[pb.String()]; ok {
				matches[rr.pattern] = &Match{RegisteredPattern: rr}
			}
		}
	}

	if !foundFullStatic {
		// For the catch-all pattern (e.g., "/*"), handle it specially
		if rr, ok := m.dynamicPatterns[m.catchAllPattern]; ok {
			matches[m.catchAllPattern] = &Match{
				RegisteredPattern: rr,
				SplatValues:       realSegments,
			}
		}

		// DFS for the rest of the matches
		params := make(Params)
		m.dfsNestedMatches(m.rootNode, realSegments, 0, params, matches)
	}

	// if there are multiple matches and a catch-all, remove the catch-all
	if _, ok := matches[m.catchAllPattern]; ok {
		if len(matches) > 1 {
			delete(matches, m.catchAllPattern)
		}
	}

	if len(matches) < 2 {
		return flattenAndSortMatches(matches)
	}

	var longestSegmentLen int
	longestSegmentMatches := make(matchesMap)
	for _, match := range matches {
		if len(match.segments) > longestSegmentLen {
			longestSegmentLen = len(match.segments)
		}
	}
	for _, match := range matches {
		if len(match.segments) == longestSegmentLen {
			longestSegmentMatches[match.lastSegType] = match
		}
	}

	// if there is any splat or index with a segment length shorter than longest segment length, remove it
	for pattern, match := range matches {
		if len(match.segments) < longestSegmentLen {
			if match.lastSegIsNonRootSplat || match.lastSegIsNestedIndex {
				delete(matches, pattern)
			}
		}
	}

	if len(matches) < 2 {
		return flattenAndSortMatches(matches)
	}

	// if the longest segment length items are (1) dynamic, (2) splat, or (3) index, remove them as follows:
	// - if the realSegmentLen equals the longest segment length, prioritize dynamic, then splat, and always remove index
	// - if the realSegmentLen is greater than the longest segment length, prioritize splat, and always remove dynamic and index
	if len(longestSegmentMatches) > 1 {
		if match, indexExists := longestSegmentMatches[segTypes.index]; indexExists {
			delete(matches, match.pattern)
		}

		_, dynamicExists := longestSegmentMatches[segTypes.dynamic]
		_, splatExists := longestSegmentMatches[segTypes.splat]

		if len(realSegments) == longestSegmentLen && dynamicExists && splatExists {
			delete(matches, longestSegmentMatches[segTypes.splat].pattern)
		}
		if len(realSegments) > longestSegmentLen && splatExists && dynamicExists {
			delete(matches, longestSegmentMatches[segTypes.dynamic].pattern)
		}
	}

	return flattenAndSortMatches(matches)
}

func (m *Matcher) dfsNestedMatches(
	node *segmentNode,
	segments []string,
	depth int,
	params Params,
	matches matchesMap,
) {
	if len(node.pattern) > 0 {
		if rp := m.dynamicPatterns[node.pattern]; rp != nil {
			// Don't process the ultimate catch-all here
			if node.pattern != m.catchAllPattern {
				// Copy params
				paramsCopy := make(Params, len(params))
				for k, v := range params {
					paramsCopy[k] = v
				}

				var splatValues []string
				if node.nodeType == nodeSplat && depth < len(segments) {
					// For splat nodes, collect all remaining segments
					splatValues = make([]string, len(segments)-depth)
					copy(splatValues, segments[depth:])
				}

				match := &Match{
					RegisteredPattern: rp,
					Params:            paramsCopy,
					SplatValues:       splatValues,
				}
				matches[node.pattern] = match

				// Check for nested index signifier if we're at the exact depth
				if depth == len(segments) {
					indexPattern := node.pattern + m.slashNestedIndexSignifier
					if rp, ok := m.dynamicPatterns[indexPattern]; ok {
						matches[indexPattern] = &Match{
							RegisteredPattern: rp,
							Params:            paramsCopy,
						}
					}
				}
			}
		}
	}

	// If we've consumed all segments, stop
	if depth >= len(segments) {
		return
	}

	seg := segments[depth]

	// Try static children
	if node.children != nil {
		if child, ok := node.children[seg]; ok {
			m.dfsNestedMatches(child, segments, depth+1, params, matches)
		}
	}

	// Try dynamic/splat children
	for _, child := range node.dynChildren {
		switch child.nodeType {
		case nodeDynamic:
			// Backtracking pattern for dynamic
			oldVal, hadVal := params[child.paramName]
			params[child.paramName] = seg

			m.dfsNestedMatches(child, segments, depth+1, params, matches)

			if hadVal {
				params[child.paramName] = oldVal
			} else {
				delete(params, child.paramName)
			}

		case nodeSplat:
			// For splat nodes, we collect remaining segments and don't increment depth
			m.dfsNestedMatches(child, segments, depth, params, matches)
		}
	}
}

func flattenAndSortMatches(matches matchesMap) ([]*Match, bool) {
	var results []*Match
	for _, match := range matches {
		results = append(results, match)
	}

	slices.SortStableFunc(results, func(i, j *Match) int {
		// if any match is an index, it should be last
		if i.lastSegIsNestedIndex {
			return 1
		}
		if j.lastSegIsNestedIndex {
			return -1
		}

		// else sort by segment length
		return len(i.segments) - len(j.segments)
	})

	return results, len(results) > 0
}

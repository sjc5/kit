package router

func (m *matcher) FindBestMatch(realPath string) (*Match, bool) {
	// Fast path: exact static match
	if rr, ok := m.staticPatterns[realPath]; ok {
		return &Match{RegisteredPattern: rr}, true
	}

	segments := ParseSegments(realPath)

	// For the DFS we track the best match in these pointers:
	var best *Match
	var bestScore uint16

	// Reuse a single Params map for backtracking:
	params := make(Params, 4)

	m.dfsBest(
		m.rootNode,
		segments,
		0, // depth
		0, // initial score
		params,
		&best,
		&bestScore,
	)
	return best, best != nil
}

// dfsBest does a depth-first search through the trie, tracking the best match found.
func (m *matcher) dfsBest(
	node *segmentNode,
	segments []string,
	depth int,
	score uint16,
	params Params, // reused param map
	best **Match,
	bestScore *uint16,
) {
	// 1. If this node itself is a terminal route, see if it qualifies.
	if len(node.pattern) > 0 {
		if rp := m.dynamicPatterns[node.pattern]; rp != nil {
			// We accept it if we've consumed all segments OR it's a splat node.
			if depth == len(segments) || node.nodeType == nodeSplat {
				// Copy params
				copiedParams := make(Params, len(params))
				for k, v := range params {
					copiedParams[k] = v
				}

				var splat []string
				// If it's the splat node and we haven't consumed everything
				// we capture what's left.
				if node.nodeType == nodeSplat && depth < len(segments) {
					splat = segments[depth:]
				}

				m := &Match{
					RegisteredPattern: rp,
					Params:            copiedParams,
					SplatValues:       splat,
					score:             score,
				}
				if *best == nil || m.score > (*best).score {
					*best = m
					*bestScore = m.score
				}
			}
		}
	}

	// 2. If we've consumed all path segments, we cannot descend further.
	if depth >= len(segments) {
		return
	}

	seg := segments[depth]

	// 3. Try a static child
	if node.children != nil {
		if child, ok := node.children[seg]; ok {
			m.dfsBest(child, segments, depth+1, score+scoreStaticMatch, params, best, bestScore)
		}
	}

	// 4. Try dynamic/splat children
	for _, child := range node.dynChildren {
		switch child.nodeType {
		case nodeDynamic:
			// Backtracking pattern for dynamic
			oldVal, hadVal := params[child.paramName]
			params[child.paramName] = seg

			m.dfsBest(child, segments, depth+1, score+scoreDynamic, params, best, bestScore)

			if hadVal {
				params[child.paramName] = oldVal
			} else {
				delete(params, child.paramName)
			}

		case nodeSplat:
			// Capture whatever is left in the path
			leftover := segments[depth:]
			splatScore := score + scoreSplat

			// If this child node itself is a route, record an immediate match.
			// (Often the child node is the final route pattern for a splat.)
			if len(child.pattern) > 0 {
				if rp := m.dynamicPatterns[child.pattern]; rp != nil {
					copiedParams := make(Params, len(params))
					for k, v := range params {
						copiedParams[k] = v
					}
					m := &Match{
						RegisteredPattern: rp,
						Params:            copiedParams,
						SplatValues:       leftover,
						score:             splatScore,
					}
					if *best == nil || m.score > (*best).score {
						*best = m
						*bestScore = m.score
					}
				}
			}

			// Then recurse with depth jumped to the end, so we do not re-match leftover segments.
			// This allows deeper nodes under the splat if you want them (often you do not).
			m.dfsBest(child, segments, len(segments), splatScore, params, best, bestScore)
		}
	}
}

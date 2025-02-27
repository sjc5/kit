package matcher

func (m *Matcher) FindBestMatch(realPath string) (*Match, bool) {
	if rr, ok := m.staticPatterns[realPath]; ok {
		return &Match{RegisteredPattern: rr}, true
	}

	segments := ParseSegments(realPath)

	best := new(Match)
	var bestScore uint16
	foundMatch := false

	m.dfsBest(m.rootNode, segments, 0, 0, best, &bestScore, &foundMatch)

	if !foundMatch {
		return nil, false
	}

	if best.numberOfDynamicParamSegs > 0 {
		params := make(Params, best.numberOfDynamicParamSegs)
		for i, seg := range best.segments {
			if seg.segType == segTypes.dynamic {
				params[seg.value[1:]] = segments[i]
			}
		}
		best.Params = params
	}

	if best.pattern == m.catchAllPattern || best.lastSegIsNonRootSplat {
		best.SplatValues = segments[len(best.segments)-1:]
	}

	return best, true
}

func (m *Matcher) dfsBest(
	node *segmentNode,
	segments []string,
	depth int,
	score uint16,
	best *Match,
	bestScore *uint16,
	foundMatch *bool,
) {
	if len(node.pattern) > 0 {
		if rp, ok := m.dynamicPatterns[node.pattern]; ok {
			if depth == len(segments) || node.nodeType == nodeSplat {
				if !*foundMatch || score > *bestScore {
					best.RegisteredPattern = rp
					best.score = score
					*bestScore = score
					*foundMatch = true
				}
			}
		}
	}

	if depth >= len(segments) {
		return
	}

	if node.children != nil {
		if child, ok := node.children[segments[depth]]; ok {
			m.dfsBest(child, segments, depth+1, score+scoreStaticMatch, best, bestScore, foundMatch)

			if *foundMatch && depth+1 == len(segments) && child.pattern != "" {
				return
			}
		}
	}

	for _, child := range node.dynChildren {
		switch child.nodeType {
		case nodeDynamic:
			m.dfsBest(child, segments, depth+1, score+scoreDynamic, best, bestScore, foundMatch)

		case nodeSplat:
			if len(child.pattern) > 0 {
				if rp := m.dynamicPatterns[child.pattern]; rp != nil {
					newScore := score + scoreSplat
					if !*foundMatch || newScore > *bestScore {
						best.RegisteredPattern = rp
						best.score = newScore
						*bestScore = newScore
						*foundMatch = true
					}
				}
			}
		}
	}
}

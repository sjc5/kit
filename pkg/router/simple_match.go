package router

// only need to run this on dynamic routes
// you can find static routes by just checking the map
func (router *RouterBest) SimpleMatch(pattern, realPath string, nested bool, withIndex bool) (*Match, bool) {
	rr, ok := router.DynamicRegisteredRoutes[pattern]
	if !ok {
		return nil, false
	}

	if withIndex {
		if !rr.LastSegmentIsIndex() {
			return nil, false
		}
		realPath += "/" + router.NestedIndexSignifier
	}

	patternSegmentsLen := len(rr.Segments)
	realSegments := ParseSegments(realPath)
	realSegmentsLen := len(realSegments)

	if !nested && realSegmentsLen > patternSegmentsLen && !rr.LastSegmentIsSplat() {
		return nil, false
	}

	if patternSegmentsLen > realSegmentsLen+1 || (patternSegmentsLen > realSegmentsLen && !rr.LastSegmentIsIndex()) {
		return nil, false
	}

	var params Params
	var score int
	var splatValues []string

	for i, patternSegment := range rr.Segments {
		if i >= realSegmentsLen {
			return nil, false
		}

		isLastSegment := i == patternSegmentsLen-1

		if isLastSegment && patternSegment.Type == SegmentTypes.Index {
			break
		}

		switch {
		case patternSegment.Value == realSegments[i]: // Exact match
			score += scoreStaticMatch
		case patternSegment.Type == SegmentTypes.Dynamic: // Dynamic parameter
			score += scoreDynamic
			if params == nil {
				params = make(Params)
			}
			params[patternSegment.Value[1:]] = realSegments[i]
		case patternSegment.Type == SegmentTypes.Splat: // Splat segment
			score += scoreSplat
			if isLastSegment {
				if withIndex {
					splatValues = realSegments[i : realSegmentsLen-1]
				} else {
					splatValues = realSegments[i:]
				}
			}

		default:
			return nil, false
		}
	}

	results := &Match{
		RegisteredRoute: rr,
		Params:          params,
		Score:           score,
	}

	if splatValues != nil {
		results.SplatValues = splatValues
	}

	return results, true
}

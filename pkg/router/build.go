package router

import "strings"

func (router *RouterBest) AddRoute(pattern string) {
	router.MakeDataStructuresIfNeeded()

	rawSegments := ParseSegments(pattern)

	segments := make([]*Segment, 0, len(rawSegments))
	for _, segment := range rawSegments {
		if router.ShouldExcludeSegmentFunc != nil && router.ShouldExcludeSegmentFunc(segment) {
			continue
		}
		segments = append(segments, &Segment{
			Value: segment,
			Type:  router.getSegmentType(segment),
		})
	}

	registeredRoute := &RegisteredRoute{Pattern: pattern, Segments: segments}

	totalScore, isStatic := getTotalScoreAndIsStatic(segments)

	if isStatic {
		router.StaticRegisteredRoutes[pattern] = registeredRoute
		router.trie.staticRoutes[pattern] = totalScore
		return
	}

	router.DynamicRegisteredRoutes[pattern] = registeredRoute

	current := router.trie.root
	var nodeScore int

	for i, segment := range segments {
		child := current.findOrCreateChild(segment.Value)
		switch {
		case segment.Type == SegmentTypes.Splat:
			nodeScore += scoreSplat
		case segment.Type == SegmentTypes.Dynamic:
			nodeScore += scoreDynamic
		default:
			nodeScore += scoreStaticMatch
		}

		if i == len(segments)-1 {
			child.finalScore = nodeScore
			child.pattern = pattern
		}

		current = child
	}
}

func (router *RouterBest) getSegmentType(segment string) SegmentType {
	switch {
	case segment == router.NestedIndexSignifier:
		return SegmentTypes.Index
	case segment == "$":
		return SegmentTypes.Splat
	case len(segment) > 0 && segment[0] == '$':
		return SegmentTypes.Dynamic
	default:
		return SegmentTypes.Static
	}
}

func segmentsToPattern(segments []*Segment) string {
	var sb strings.Builder
	sb.WriteString("/")
	for i, segment := range segments {
		if i > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(segment.Value)
	}
	return sb.String()
}

func getTotalScoreAndIsStatic(segments []*Segment) (int, bool) {
	var totalScore int
	isStatic := true

	if len(segments) > 0 {
		for _, segment := range segments {
			switch segment.Type {
			case SegmentTypes.Splat:
				totalScore += scoreSplat
				isStatic = false
			case SegmentTypes.Dynamic:
				totalScore += scoreDynamic
				isStatic = false
			default:
				totalScore += scoreStaticMatch
			}
		}
	} else {
		totalScore = scoreStaticMatch
	}

	return totalScore, isStatic
}

func (router *RouterBest) MakeDataStructuresIfNeeded() {
	if router.StaticRegisteredRoutes == nil {
		router.StaticRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
	if router.DynamicRegisteredRoutes == nil {
		router.DynamicRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
	if router.trie == nil {
		router.trie = makeTrie()
	}
}

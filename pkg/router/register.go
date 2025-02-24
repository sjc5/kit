package router

// Note -- should we validate that there are no two competing dynamic segments in otherwise matching patterns?

type RegisteredPattern struct {
	pattern     string
	segments    []*segment
	handler     Handler
	middlewares []Middleware

	// pre-computed helpers
	lastSegType           segType
	lastSegIsNonRootSplat bool
	lastSegIsIndex        bool
}

func (rp *RegisteredPattern) AddMiddleware(middleware Middleware) *RegisteredPattern {
	rp.middlewares = append(rp.middlewares, middleware)
	return rp
}

func (rp *RegisteredPattern) SetHandler(handler Handler) *RegisteredPattern {
	rp.handler = handler
	return rp
}

func (m *matcher) RegisterPattern(pattern string) *RegisteredPattern {
	rawSegments := ParseSegments(pattern)
	segments := make([]*segment, 0, len(rawSegments))

	for _, seg := range rawSegments {
		if m.shouldExcludeSegmentFunc != nil && m.shouldExcludeSegmentFunc(seg) {
			continue
		}
		segments = append(segments, &segment{
			value:   seg,
			segType: m.getSegmentType(seg),
		})
	}

	segLen := len(segments)
	var lastType segType
	if segLen > 0 {
		lastType = segments[segLen-1].segType
	}

	rp := &RegisteredPattern{
		pattern:               pattern,
		segments:              segments,
		lastSegType:           lastType,
		lastSegIsNonRootSplat: lastType == segTypes.splat && segLen > 1,
		lastSegIsIndex:        lastType == segTypes.index,
	}

	if getIsStatic(segments) {
		m.staticPatterns[pattern] = rp
		return rp
	}

	m.dynamicPatterns[pattern] = rp

	current := m.rootNode
	var nodeScore int

	for i, segment := range segments {
		child := current.findOrCreateChild(segment.value)
		switch {
		case segment.segType == segTypes.splat:
			nodeScore += scoreSplat
		case segment.segType == segTypes.dynamic:
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

	return rp
}

func (m *matcher) getSegmentType(segment string) segType {
	switch {
	case segment == m.nestedIndexSignifier:
		return segTypes.index
	case segment == "$":
		return segTypes.splat
	case len(segment) > 0 && segment[0] == '$':
		return segTypes.dynamic
	default:
		return segTypes.static
	}
}

func getIsStatic(segments []*segment) bool {
	if len(segments) > 0 {
		for _, segment := range segments {
			switch segment.segType {
			case segTypes.splat:
				return false
			case segTypes.dynamic:
				return false
			}
		}
	}
	return true
}

type segmentNode struct {
	pattern     string
	nodeType    uint8
	children    map[string]*segmentNode
	dynChildren []*segmentNode
	paramName   string
	finalScore  int
}

// findOrCreateChild finds or creates a child node for a segment
func (n *segmentNode) findOrCreateChild(segment string) *segmentNode {
	if segment == "$" || (len(segment) > 0 && segment[0] == '$') {
		for _, child := range n.dynChildren {
			if child.paramName == segment[1:] {
				return child
			}
		}
		return n.addDynamicChild(segment)
	}

	if n.children == nil {
		n.children = make(map[string]*segmentNode)
	}
	if child, exists := n.children[segment]; exists {
		return child
	}
	child := &segmentNode{nodeType: nodeStatic}
	n.children[segment] = child
	return child
}

// addDynamicChild creates a new dynamic or splat child node
func (n *segmentNode) addDynamicChild(segment string) *segmentNode {
	child := &segmentNode{}
	if segment == "$" {
		child.nodeType = nodeSplat
	} else {
		child.nodeType = nodeDynamic
		child.paramName = segment[1:]
	}
	n.dynChildren = append(n.dynChildren, child)
	return child
}

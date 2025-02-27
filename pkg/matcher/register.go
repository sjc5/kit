package matcher

// Note -- should we validate that there are no two competing dynamic segments in otherwise matching patterns?

const (
	nodeStatic       uint8 = 0
	nodeDynamic      uint8 = 1
	nodeSplat        uint8 = 2
	scoreStaticMatch       = 3
	scoreDynamic           = 2
	scoreSplat             = 1
)

type RegisteredPattern struct {
	pattern                  string
	segments                 []*segment
	lastSegType              segType
	lastSegIsNonRootSplat    bool
	lastSegIsNestedIndex     bool
	numberOfDynamicParamSegs uint8
}

func (rp *RegisteredPattern) Pattern() string {
	return rp.pattern
}

type segment struct {
	value   string
	segType segType
}

var segTypes = struct {
	splat   segType
	static  segType
	dynamic segType
	index   segType
}{
	splat:   "splat",
	static:  "static",
	dynamic: "dynamic",
	index:   "index",
}

func (m *Matcher) RegisterPattern(pattern string) *RegisteredPattern {
	rawSegments := ParseSegments(pattern)
	segments := make([]*segment, 0, len(rawSegments))

	var numberOfDynamicParamSegs uint8

	for _, seg := range rawSegments {
		segType := m.getSegmentType(seg)
		if segType == segTypes.dynamic {
			numberOfDynamicParamSegs++
		}

		segments = append(segments, &segment{
			value:   seg,
			segType: segType,
		})
	}

	segLen := len(segments)
	var lastType segType
	if segLen > 0 {
		lastType = segments[segLen-1].segType
	}

	rp := &RegisteredPattern{
		pattern:                  pattern,
		segments:                 segments,
		lastSegType:              lastType,
		lastSegIsNonRootSplat:    lastType == segTypes.splat && segLen > 1,
		lastSegIsNestedIndex:     lastType == segTypes.index,
		numberOfDynamicParamSegs: numberOfDynamicParamSegs,
	}

	if getIsStatic(segments) {
		m.staticPatterns[pattern] = rp
		return rp
	}

	m.dynamicPatterns[pattern] = rp

	current := m.rootNode
	var nodeScore int

	for i, segment := range segments {
		child := current.findOrCreateChild(segment.value, m.splatSegmentRune, m.dynamicParamPrefixRune)
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

func (m *Matcher) getSegmentType(segment string) segType {
	switch {
	case segment == m.nestedIndexSignifier:
		return segTypes.index
	case len(segment) == 1 && segment == string(m.splatSegmentRune):
		return segTypes.splat
	case len(segment) > 0 && segment[0] == byte(m.dynamicParamPrefixRune):
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
func (n *segmentNode) findOrCreateChild(segment string, splatRune rune, dynRune rune) *segmentNode {
	if segment == string(splatRune) || (len(segment) > 0 && rune(segment[0]) == dynRune) {
		for _, child := range n.dynChildren {
			if child.paramName == segment[1:] {
				return child
			}
		}
		return n.addDynamicChild(segment, splatRune)
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
func (n *segmentNode) addDynamicChild(segment string, splatRune rune) *segmentNode {
	child := &segmentNode{}
	if segment == string(splatRune) {
		child.nodeType = nodeSplat
	} else {
		child.nodeType = nodeDynamic
		child.paramName = segment[1:]
	}
	n.dynChildren = append(n.dynChildren, child)
	return child
}

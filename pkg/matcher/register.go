package matcher

import (
	"fmt"
	"log"
	"strings"
)

// Note -- __TODO should we validate that there are no two competing dynamic segments in otherwise matching patterns?

const (
	nodeStatic       uint8 = 0
	nodeDynamic      uint8 = 1
	nodeSplat        uint8 = 2
	scoreStaticMatch       = 2
	scoreDynamic           = 1
)

type RegisteredPattern struct {
	originalPattern          string
	normalizedPattern        string
	normalizedSegments       []*segment
	lastSegType              segType
	lastSegIsNonRootSplat    bool
	lastSegIsIndex           bool
	numberOfDynamicParamSegs uint8
}

func (rp *RegisteredPattern) NormalizedPattern() string {
	return rp.normalizedPattern
}

func (rp *RegisteredPattern) NormalizedSegments() []*segment {
	return rp.normalizedSegments
}

func (rp *RegisteredPattern) OriginalPattern() string {
	return rp.originalPattern
}

func HasTrailingSlash(pattern string) bool {
	return len(pattern) > 0 && pattern[len(pattern)-1] == '/'
}

func HasLeadingSlash(pattern string) bool {
	return len(pattern) > 0 && pattern[0] == '/'
}

func JoinPatterns(rp *RegisteredPattern, pattern string) string {
	var sb strings.Builder
	base := rp.normalizedPattern
	sb.WriteString(base)

	patternHasLeadingSlash := HasLeadingSlash(pattern)

	if HasTrailingSlash(base) && patternHasLeadingSlash {
		pattern = pattern[1:]
	} else if !patternHasLeadingSlash {
		sb.WriteString("/")
	}

	sb.WriteString(pattern)

	return sb.String()
}

type segment struct {
	normalizedVal string
	segType       segType
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

func getAppropriateWarningMsg(pattern string, usingExplicitIndexSegment bool) string {
	base := fmt.Sprintf("WARN: Pattern '%s' is already registered.", pattern)
	if usingExplicitIndexSegment {
		return base + " When you use an explicit index segment, trailing slashes are ignored, which may be the reason for your effectively duplicated patterns."
	}
	return base
}

func (m *Matcher) Log(msg string) {
	if !m.quiet {
		log.Println(msg)
	}
}

func (m *Matcher) NormalizePattern(originalPattern string) *RegisteredPattern {
	normalizedPattern := originalPattern

	// if using an index sig
	if m.usingExplicitIndexSegment {
		// ignore trailing slashes
		if strings.HasSuffix(normalizedPattern, "/") {
			m.Log(fmt.Sprintf("WARN: Trailing slashes are ignored when using an explicit index segment. Pattern '%s' will be normalized without the trailing slash.", originalPattern))
			normalizedPattern = strings.TrimRight(normalizedPattern, "/")
		}
		// if is an idx route, clear the sig, but leave the trailing slash
		if strings.HasSuffix(normalizedPattern, m.slashIndexSegment) {
			normalizedPattern = strings.TrimRight(normalizedPattern, m.explicitIndexSegment)
		}

		// Now patterns with a trailing slash are index routes, and those without a trailing
		// slash are non-index routes. This means that the normalized pattern for the "true"
		// root would be an empty string, whereas the normalized pattern for the index route
		// would be a single slash.
	}

	rawSegments := ParseSegments(normalizedPattern)
	segments := make([]*segment, 0, len(rawSegments))

	var numberOfDynamicParamSegs uint8

	for _, seg := range rawSegments {
		normalizedVal := seg

		segType := m.getSegmentTypeAssumeNormalized(seg)
		if segType == segTypes.dynamic {
			numberOfDynamicParamSegs++
			normalizedVal = ":" + seg[1:]
		}
		if segType == segTypes.splat {
			normalizedVal = "*"
		}

		segments = append(segments, &segment{
			normalizedVal: normalizedVal,
			segType:       segType,
		})
	}

	segLen := len(segments)
	var lastType segType
	if segLen > 0 {
		lastType = segments[segLen-1].segType
	}

	var finalNormalizedPatternBuilder strings.Builder
	finalNormalizedPatternBuilder.WriteString("/")
	for i, seg := range segments {
		finalNormalizedPatternBuilder.WriteString(seg.normalizedVal)
		if i < segLen-1 {
			finalNormalizedPatternBuilder.WriteString("/")
		}
	}

	finalNormalizedPattern := finalNormalizedPatternBuilder.String()

	if strings.HasSuffix(finalNormalizedPattern, "/") && lastType != segTypes.index {
		finalNormalizedPattern = strings.TrimRight(finalNormalizedPattern, "/")
	}

	return &RegisteredPattern{
		originalPattern:          originalPattern,
		normalizedPattern:        finalNormalizedPattern,
		normalizedSegments:       segments,
		lastSegType:              lastType,
		lastSegIsNonRootSplat:    lastType == segTypes.splat && segLen > 1,
		lastSegIsIndex:           lastType == segTypes.index,
		numberOfDynamicParamSegs: numberOfDynamicParamSegs,
	}
}

func (m *Matcher) RegisterPattern(originalPattern string) *RegisteredPattern {
	n := m.NormalizePattern(originalPattern)

	if _, alreadyRegistered := m.staticPatterns[n.normalizedPattern]; alreadyRegistered {
		m.Log(getAppropriateWarningMsg(originalPattern, m.usingExplicitIndexSegment))
	}
	if _, alreadyRegistered := m.dynamicPatterns[n.normalizedPattern]; alreadyRegistered {
		m.Log(getAppropriateWarningMsg(originalPattern, m.usingExplicitIndexSegment))
	}

	if getIsStatic(n.normalizedSegments) {
		m.staticPatterns[n.normalizedPattern] = n
		return n
	}

	m.dynamicPatterns[n.normalizedPattern] = n

	current := m.rootNode
	var nodeScore int

	for i, segment := range n.normalizedSegments {
		child := current.findOrCreateChild(segment.normalizedVal)
		switch {
		case segment.segType == segTypes.dynamic:
			nodeScore += scoreDynamic
		case segment.segType != segTypes.splat:
			nodeScore += scoreStaticMatch
		}

		if i == len(n.normalizedSegments)-1 {
			child.finalScore = nodeScore
			child.pattern = n.normalizedPattern
		}

		current = child
	}

	return n
}

func (m *Matcher) getSegmentTypeAssumeNormalized(segment string) segType {
	switch {
	case segment == "":
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
func (n *segmentNode) findOrCreateChild(segment string) *segmentNode {
	if segment == "*" || (len(segment) > 0 && rune(segment[0]) == ':') {
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
	if segment == "*" {
		child.nodeType = nodeSplat
	} else {
		child.nodeType = nodeDynamic
		child.paramName = segment[1:]
	}
	n.dynChildren = append(n.dynChildren, child)
	return child
}

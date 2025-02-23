package router

import (
	"slices"
	"strings"
)

const USE_TRIE = true

const (
	scoreStaticMatch = 3
	scoreDynamic     = 2
	scoreSplat       = 1
)

type Params = map[string]string

type Match struct {
	*RegisteredRoute

	Params      Params
	SplatValues []string
	Score       int
}

type SegmentType = string

var SegmentTypes = struct {
	Splat   SegmentType
	Static  SegmentType
	Dynamic SegmentType
	Index   SegmentType
}{
	Splat:   "splat",
	Static:  "static",
	Dynamic: "dynamic",
	Index:   "index",
}

type Segment struct {
	Value string
	Type  string
}

type RegisteredRoute struct {
	Pattern  string
	Segments []*Segment
}

func (rr *RegisteredRoute) GetLastSegmentType() SegmentType {
	return rr.Segments[len(rr.Segments)-1].Type
}

func (rr *RegisteredRoute) LastSegmentIsIndex() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Index
}

func (rr *RegisteredRoute) LastSegmentIsSplat() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Splat
}

func (rr *RegisteredRoute) LastSegmentIsUltimateCatch() bool {
	return rr.LastSegmentIsSplat() && len(rr.Segments) == 1
}

func (rr *RegisteredRoute) LastSegmentIsStaticLayout() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Static
}

func (rr *RegisteredRoute) LastSegmentIsNonUltimateSplat() bool {
	return rr.LastSegmentIsSplat() && len(rr.Segments) > 1
}

func (rr *RegisteredRoute) LastSegmentIsDynamicLayout() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Dynamic
}

func (rr *RegisteredRoute) LastSegmentIsIndexPrecededByDynamic() bool {
	segmentsLen := len(rr.Segments)

	return rr.LastSegmentIsIndex() &&
		segmentsLen >= 2 &&
		rr.Segments[segmentsLen-2].Type == SegmentTypes.Dynamic
}

func (rr *RegisteredRoute) IndexAdjustedPatternLen() int {
	if rr.LastSegmentIsIndex() {
		return len(rr.Segments) - 1
	}
	return len(rr.Segments)
}

type Pattern = string

type RouterBest struct {
	NestedIndexSignifier string
	// e.g., "_index"

	ShouldExcludeSegmentFunc func(segment string) bool
	// e.g., return strings.HasPrefix(segment, "__")

	trie *trie

	StaticRegisteredRoutes  map[Pattern]*RegisteredRoute
	DynamicRegisteredRoutes map[Pattern]*RegisteredRoute
}

// Note -- should we validate that there are no two competing dynamic segments in otherwise matching routes?

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
	if router.trie == nil {
		router.trie = makeTrie()
	}
	if router.StaticRegisteredRoutes == nil {
		router.StaticRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
	if router.DynamicRegisteredRoutes == nil {
		router.DynamicRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
}

func (router *RouterBest) FindBestMatch(realPath string) (*Match, bool) {
	router.MakeDataStructuresIfNeeded()

	// fast path if totally static
	if rr, ok := router.StaticRegisteredRoutes[realPath]; ok {
		return &Match{RegisteredRoute: rr}, true
	}

	if USE_TRIE {
		realSegments := ParseSegments(realPath)
		traverse, getMatches := router.makeTraverseFunc(realSegments, false)
		traverse(router.trie.root, 0, 0)
		matches := getMatches()
		if len(matches) == 0 {
			return nil, false
		}
		bestMatch := matches[0]
		return bestMatch, bestMatch != nil
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
}

type MatchesMap = map[string]*Match

func (router *RouterBest) FindAllMatches(realPath string) ([]*Match, bool) {
	realSegments := ParseSegments(realPath)
	matches := make(MatchesMap)

	if USE_TRIE {
		realSegments := ParseSegments(realPath)
		traverse, getMatches := router.makeTraverseFunc(realSegments, true)
		traverse(router.trie.root, 0, 0)
		matchesSlice := getMatches()
		if len(matchesSlice) == 0 {
			return nil, false
		}
		for _, match := range matchesSlice {
			matches[match.Pattern] = match
		}
	} else {
		if len(realSegments) == 0 {
			if rr, ok := router.StaticRegisteredRoutes["/"]; ok {
				matches[rr.Pattern] = &Match{RegisteredRoute: rr}
			}
			if rr, ok := router.StaticRegisteredRoutes["/"+router.NestedIndexSignifier]; ok {
				matches[rr.Pattern] = &Match{RegisteredRoute: rr}
			}

			return flattenMatches(matches)
		}

		var path string
		var foundFullStatic bool
		for i := 0; i < len(realSegments); i++ {
			path += "/" + realSegments[i]
			if rr, ok := router.StaticRegisteredRoutes[path]; ok {
				matches[rr.Pattern] = &Match{RegisteredRoute: rr}
				if i == len(realSegments)-1 {
					foundFullStatic = true
				}
			}
			if i == len(realSegments)-1 {
				if rr, ok := router.StaticRegisteredRoutes[path+"/"+router.NestedIndexSignifier]; ok {
					matches[rr.Pattern] = &Match{RegisteredRoute: rr}
				}
			}
		}

		if !foundFullStatic {
			for pattern := range router.DynamicRegisteredRoutes {
				if match, ok := router.SimpleMatch(pattern, realPath, true, false); ok {
					matches[pattern] = match
				}
				if match, ok := router.SimpleMatch(pattern, realPath, true, true); ok {
					matches[pattern] = match
				}
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

///////////////////////////////////////////
///// TRIE
///////////////////////////////////////////

const (
	nodeStatic  uint8 = 0
	nodeDynamic uint8 = 1
	nodeSplat   uint8 = 2
)

type segmentNode struct {
	pattern     string
	nodeType    uint8
	children    map[string]*segmentNode
	dynChildren []*segmentNode
	paramName   string
	finalScore  int
}

type trie struct {
	root         *segmentNode
	staticRoutes map[string]int
}

func makeTrie() *trie {
	return &trie{
		root:         &segmentNode{},
		staticRoutes: make(map[string]int),
	}
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

type traverseFunc func(node *segmentNode, depth int, score int)

func (router *RouterBest) makeTraverseFunc(segments []string, findAllMatches bool) (traverseFunc, func() []*Match) {
	matches := make(MatchesMap)
	currentParams := make(Params)

	// Determine if we're in nested mode based on NestedIndexSignifier
	isNested := router.NestedIndexSignifier != ""

	// Check for root index (nested mode only)
	if len(segments) == 0 && isNested {
		if rr, ok := router.StaticRegisteredRoutes["/"+router.NestedIndexSignifier]; ok {
			matches[rr.Pattern] = &Match{RegisteredRoute: rr}
			return func(node *segmentNode, depth int, score int) {}, func() []*Match {
				return []*Match{matches[rr.Pattern]}
			}
		}
	}

	// Handle static routes based on mode
	if findAllMatches {
		var path string
		for i, segment := range segments {
			path += "/" + segment

			if rr, ok := router.StaticRegisteredRoutes[path]; ok {
				matches[rr.Pattern] = &Match{RegisteredRoute: rr}
			}

			// Check for index in nested mode
			if isNested && i == len(segments)-1 {
				indexPath := path + "/" + router.NestedIndexSignifier
				if rr, ok := router.StaticRegisteredRoutes[indexPath]; ok {
					matches[rr.Pattern] = &Match{RegisteredRoute: rr}
				}
			}
		}
	} else {
		fullPath := "/" + strings.Join(segments, "/")
		if rr, ok := router.StaticRegisteredRoutes[fullPath]; ok {
			matches[rr.Pattern] = &Match{RegisteredRoute: rr}
		}
	}

	var traverse traverseFunc
	traverse = func(node *segmentNode, depth int, score int) {
		if node.pattern != "" {
			rr, ok := router.DynamicRegisteredRoutes[node.pattern]
			if ok {
				// Different matching rules for nested vs non-nested
				shouldIncludeMatch := false
				if isNested {
					// In nested mode, require exact depth match unless it's a splat
					shouldIncludeMatch = findAllMatches || depth == len(segments) || node.nodeType == nodeSplat
				} else {
					// In non-nested mode, allow partial matches for dynamic segments
					shouldIncludeMatch = (findAllMatches && depth <= len(segments)) ||
						depth == len(segments) ||
						node.nodeType == nodeSplat
				}

				if shouldIncludeMatch {
					paramsCopy := make(Params)
					for k, v := range currentParams {
						paramsCopy[k] = v
					}

					var splatValues []string
					if node.nodeType == nodeSplat && depth < len(segments) {
						splatValues = make([]string, len(segments)-depth)
						copy(splatValues, segments[depth:])
					}

					match := &Match{
						RegisteredRoute: rr,
						Params:          paramsCopy,
						SplatValues:     splatValues,
						Score:           score,
					}

					// Special handling for catch-all routes
					if node.pattern == "/$" {
						if len(matches) == 0 {
							matches[node.pattern] = match
						}
					} else {
						if !findAllMatches {
							// For FindBestMatch, keep highest score
							if existing, ok := matches[node.pattern]; !ok || match.Score > existing.Score {
								matches[node.pattern] = match
							}
						} else {
							matches[node.pattern] = match

							// Check for index route in nested mode
							if isNested && depth == len(segments) {
								indexPattern := node.pattern + "/" + router.NestedIndexSignifier
								if indexRR, ok := router.DynamicRegisteredRoutes[indexPattern]; ok {
									matches[indexPattern] = &Match{
										RegisteredRoute: indexRR,
										Params:          paramsCopy,
										Score:           score,
									}
								}
							}
						}
					}
				}
			}
		}

		if depth >= len(segments) {
			return
		}

		segment := segments[depth]

		// Try static children
		if node.children != nil {
			if child, ok := node.children[segment]; ok {
				traverse(child, depth+1, score+scoreStaticMatch)
			}
		}

		// Try dynamic children
		for _, child := range node.dynChildren {
			switch child.nodeType {
			case nodeDynamic:
				currentParams[child.paramName] = segment
				traverse(child, depth+1, score+scoreDynamic)
				delete(currentParams, child.paramName)
			case nodeSplat:
				traverse(child, depth, score+scoreSplat)
			}
		}
	}

	return traverse, func() []*Match {
		results := make([]*Match, 0, len(matches))
		for _, match := range matches {
			results = append(results, match)
		}
		return results
	}
}

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

func ParseSegments(path string) []string {
	// Fast path for common cases
	if path == "" || path == "/" {
		return []string{}
	}

	// Start with a high capacity to avoid resizes
	// Most URLs have fewer than 8 segments
	var segs []string

	// Skip leading slash
	startIdx := 0
	if path[0] == '/' {
		startIdx = 1
	}

	// Maximum potential segments
	maxSegments := 0
	for i := startIdx; i < len(path); i++ {
		if path[i] == '/' {
			maxSegments++
		}
	}

	// Add one more for the final segment if path doesn't end with slash
	if len(path) > 0 && path[len(path)-1] != '/' {
		maxSegments++
	}

	if maxSegments > 0 {
		segs = make([]string, 0, maxSegments)
	}

	// Manual parsing is faster than strings.Split+TrimPrefix+TrimSuffix
	var start = startIdx

	for i := startIdx; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				segs = append(segs, path[start:i])
			}
			start = i + 1
		}
	}

	// Add final segment
	if start < len(path) {
		segs = append(segs, path[start:])
	}

	return segs
}

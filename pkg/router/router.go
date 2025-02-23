package router

import (
	"slices"
)

const (
	scoreStaticMatch = 3
	scoreDynamic     = 2
	scoreSplat       = 1

	implOld     = "old"
	implNonTrie = "non-trie"
	implTrie    = "trie"
)

type Router struct {
	NestedIndexSignifier     string                    // e.g., "_index"
	ShouldExcludeSegmentFunc func(segment string) bool // e.g., return strings.HasPrefix(segment, "__")
	Impl                     string
	StaticRegisteredRoutes   map[Pattern]*RegisteredRoute
	DynamicRegisteredRoutes  map[Pattern]*RegisteredRoute
	trie                     *trie

	// registeredRoutes_OldNestedImpl []*RegisteredRoute_OldNestedImpl
}

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

	// Pre-computed fields
	segmentLen         int
	lastSegType        SegmentType
	isUltimateCatch    bool
	isNonUltimateSplat bool
	isDynamicLayout    bool
	isStaticLayout     bool
	isIndex            bool
}

type Pattern = string

// Note -- should we validate that there are no two competing dynamic segments in otherwise matching routes?

func (router *Router) AddRoute(pattern string) {
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

	segLen := len(segments)
	var lastType SegmentType
	if segLen > 0 {
		lastType = segments[segLen-1].Type
	}

	if router.Impl == implOld {
		for _, segment := range segments {
			if segment.Value == "_index" {
				segment.Value = ""
			}
		}
		if len(segments) == 0 {
			segments = append(segments, &Segment{Value: "", Type: SegmentTypes.Index})
		}
	}

	rr := &RegisteredRoute{
		Pattern:  pattern,
		Segments: segments,

		// Pre-compute all the commonly checked properties
		segmentLen:         segLen,
		lastSegType:        lastType,
		isUltimateCatch:    lastType == SegmentTypes.Splat && segLen == 1,
		isNonUltimateSplat: lastType == SegmentTypes.Splat && segLen > 1,
		isDynamicLayout:    lastType == SegmentTypes.Dynamic,
		isStaticLayout:     lastType == SegmentTypes.Static,
		isIndex:            lastType == SegmentTypes.Index,
	}

	totalScore, isStatic := getTotalScoreAndIsStatic(segments)

	if isStatic {
		router.StaticRegisteredRoutes[pattern] = rr
		if router.Impl == implTrie {
			router.trie.staticRoutes[pattern] = totalScore
		}
		return
	}

	router.DynamicRegisteredRoutes[pattern] = rr

	if router.Impl == implTrie {
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
}

func (router *Router) getSegmentType(segment string) SegmentType {
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

func (router *Router) MakeDataStructuresIfNeeded() {
	if router.Impl == implTrie && router.trie == nil {
		router.trie = makeTrie()
	}
	if router.StaticRegisteredRoutes == nil {
		router.StaticRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
	if router.DynamicRegisteredRoutes == nil {
		router.DynamicRegisteredRoutes = make(map[string]*RegisteredRoute)
	}
}

func (router *Router) FindBestMatch(realPath string) (*Match, bool) {
	router.MakeDataStructuresIfNeeded()

	// Fast path: exact static match
	if rr, ok := router.StaticRegisteredRoutes[realPath]; ok {
		return &Match{RegisteredRoute: rr}, true
	}

	if router.Impl == implTrie {
		segments := ParseSegments(realPath)

		// For the DFS we track the best match in these pointers:
		var best *Match
		var bestScore int

		// Reuse a single Params map for backtracking:
		params := make(Params, 4)

		dfsBest(
			router.trie.root,
			segments,
			0, // depth
			0, // initial score
			params,
			router.DynamicRegisteredRoutes,
			&best,
			&bestScore,
		)
		return best, best != nil
	}

	// Fallback for non-trie:
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

// dfsBest does a depth-first search through the trie, tracking the best match found.
func dfsBest(
	node *segmentNode,
	segments []string,
	depth int,
	score int,
	params Params, // reused param map
	routes map[string]*RegisteredRoute,
	best **Match,
	bestScore *int,
) {
	// 1. If this node itself is a terminal route, see if it qualifies.
	if node.pattern != "" {
		if rr := routes[node.pattern]; rr != nil {
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
					RegisteredRoute: rr,
					Params:          copiedParams,
					SplatValues:     splat,
					Score:           score,
				}
				if *best == nil || m.Score > (*best).Score {
					*best = m
					*bestScore = m.Score
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
			dfsBest(child, segments, depth+1, score+scoreStaticMatch, params, routes, best, bestScore)
		}
	}

	// 4. Try dynamic/splat children
	for _, child := range node.dynChildren {
		switch child.nodeType {
		case nodeDynamic:
			// Backtracking pattern for dynamic
			oldVal, hadVal := params[child.paramName]
			params[child.paramName] = seg

			dfsBest(child, segments, depth+1, score+scoreDynamic, params, routes, best, bestScore)

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
			if child.pattern != "" {
				if rr := routes[child.pattern]; rr != nil {
					copiedParams := make(Params, len(params))
					for k, v := range params {
						copiedParams[k] = v
					}
					m := &Match{
						RegisteredRoute: rr,
						Params:          copiedParams,
						SplatValues:     leftover,
						Score:           splatScore,
					}
					if *best == nil || m.Score > (*best).Score {
						*best = m
						*bestScore = m.Score
					}
				}
			}

			// Then recurse with depth jumped to the end, so we do not re-match leftover segments.
			// This allows deeper nodes under the splat if you want them (often you do not).
			dfsBest(child, segments, len(segments), splatScore, params, routes, best, bestScore)
		}
	}
}

type MatchesMap = map[string]*Match

func (router *Router) FindAllMatches(realPath string) ([]*Match, bool) {
	router.MakeDataStructuresIfNeeded()

	if router.Impl == implOld {
		return router.getMatchingPathsInternal(realPath)
	}

	realSegments := ParseSegments(realPath)
	matches := make(MatchesMap)

	// Handle empty path case
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
		if router.Impl == implTrie {
			// First collect all static matches along the path
			path := ""
			for i, seg := range realSegments {
				path += "/" + seg
				if rr, ok := router.StaticRegisteredRoutes[path]; ok {
					matches[rr.Pattern] = &Match{RegisteredRoute: rr}
				}
				// Check for index at the last segment
				if i == len(realSegments)-1 {
					if rr, ok := router.StaticRegisteredRoutes[path+"/"+router.NestedIndexSignifier]; ok {
						matches[rr.Pattern] = &Match{RegisteredRoute: rr}
					}
				}
			}

			// For the catch-all route (/$), handle it specially
			if rr, ok := router.DynamicRegisteredRoutes["/$"]; ok {
				matches["/$"] = &Match{
					RegisteredRoute: rr,
					SplatValues:     realSegments,
				}
			}

			// DFS for the rest of the matches
			params := make(Params)
			dfsAllMatches(
				router.trie.root,
				realSegments,
				0,       // depth
				params,  // reusable params map
				matches, // collected matches
				router.DynamicRegisteredRoutes,
				router.NestedIndexSignifier,
			)
		} else {
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
			longestSegmentMatches[match.lastSegType] = match
		}
	}

	// if there is any splat or index with a segment length shorter than longest segment length, remove it
	for pattern, match := range matches {
		if len(match.Segments) < longestSegmentLen {
			if match.isNonUltimateSplat || match.isIndex {
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

func dfsAllMatches(
	node *segmentNode,
	segments []string,
	depth int,
	params Params,
	matches MatchesMap,
	routes map[string]*RegisteredRoute,
	nestedIndexSignifier string,
) {
	if node.pattern != "" {
		if rr := routes[node.pattern]; rr != nil {
			// Don't process the ultimate catch-all here
			if node.pattern != "/$" {
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
					RegisteredRoute: rr,
					Params:          paramsCopy,
					SplatValues:     splatValues,
				}
				matches[node.pattern] = match

				// Check for index route if we're at the exact depth
				if depth == len(segments) {
					indexPattern := node.pattern + "/" + nestedIndexSignifier
					if indexRR := routes[indexPattern]; indexRR != nil {
						matches[indexPattern] = &Match{
							RegisteredRoute: indexRR,
							Params:          paramsCopy,
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
			dfsAllMatches(child, segments, depth+1, params, matches, routes, nestedIndexSignifier)
		}
	}

	// Try dynamic/splat children
	for _, child := range node.dynChildren {
		switch child.nodeType {
		case nodeDynamic:
			// Backtracking pattern for dynamic
			oldVal, hadVal := params[child.paramName]
			params[child.paramName] = seg

			dfsAllMatches(child, segments, depth+1, params, matches, routes, nestedIndexSignifier)

			if hadVal {
				params[child.paramName] = oldVal
			} else {
				delete(params, child.paramName)
			}

		case nodeSplat:
			// For splat nodes, we collect remaining segments and don't increment depth
			dfsAllMatches(child, segments, depth, params, matches, routes, nestedIndexSignifier)
		}
	}
}

func flattenMatches(matches MatchesMap) ([]*Match, bool) {
	var results []*Match
	for _, match := range matches {
		results = append(results, match)
	}

	slices.SortStableFunc(results, func(i, j *Match) int {
		// if any match is an index, it should be last
		if i.isIndex {
			return 1
		}
		if j.isIndex {
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

// only need to run this on dynamic routes
// you can find static routes by just checking the map
func (router *Router) SimpleMatch(pattern, realPath string, nested bool, withIndex bool) (*Match, bool) {
	rr, ok := router.DynamicRegisteredRoutes[pattern]
	if !ok {
		return nil, false
	}

	if withIndex {
		if !rr.isIndex {
			return nil, false
		}
		realPath += "/" + router.NestedIndexSignifier
	}

	realSegments := ParseSegments(realPath)
	realSegmentsLen := len(realSegments)

	if !nested && realSegmentsLen > rr.segmentLen && !(rr.isUltimateCatch || rr.isNonUltimateSplat) {
		return nil, false
	}

	if rr.segmentLen > realSegmentsLen+1 || (rr.segmentLen > realSegmentsLen && !rr.isIndex) {
		return nil, false
	}

	var params Params
	var score int
	var splatValues []string

	for i, patternSegment := range rr.Segments {
		if i >= realSegmentsLen {
			return nil, false
		}

		isLastSegment := i == rr.segmentLen-1

		if isLastSegment && patternSegment.Type == SegmentTypes.Index {
			break
		}

		switch {
		case patternSegment.Value == realSegments[i]: // Exact match
			score += scoreStaticMatch
		case patternSegment.Type == SegmentTypes.Dynamic: // Dynamic parameter
			score += scoreDynamic
			if params == nil {
				params = make(Params, 4)
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

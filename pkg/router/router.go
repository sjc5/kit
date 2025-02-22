package router

// import (
// 	"net/http"
// 	"sort"
// 	"strings"
// )

// const (
// 	nodeStatic  uint8 = 0
// 	nodeDynamic uint8 = 1
// 	nodeSplat   uint8 = 2

// 	scoreStatic  = 3
// 	scoreDynamic = 2
// 	scoreSplat   = 1
// )

// type LastSegmentType string

// var LastSegmentTypes = struct {
// 	Splat   LastSegmentType // pattern ends in splat segment (/$)
// 	Static  LastSegmentType // pattern ends in static segment (/whatever)
// 	Dynamic LastSegmentType // pattern ends in dynamic segment (/$param)
// 	Index   LastSegmentType // pattern ends in static segment (/whatever), but signified as an index
// }{
// 	Splat:   "splat",
// 	Static:  "static",
// 	Dynamic: "dynamic",
// 	Index:   "index",
// }

// type routeInfo struct {
// 	Pattern         string
// 	Score           int
// 	LastSegmentType LastSegmentType
// }

// type segmentNode struct {
// 	nodeType    uint8
// 	children    map[string]*segmentNode
// 	dynChildren []*segmentNode
// 	paramName   string
// 	routes      []routeInfo
// }

// type staticRoute struct {
// 	score           int
// 	lastSegmentType LastSegmentType
// }

// type Router struct {
// 	root         *segmentNode
// 	staticRoutes map[string][]staticRoute
// }

// type Params = map[string]string

// type Match struct {
// 	Pattern         string
// 	Params          Params
// 	SplatSegments   []string
// 	Score           int
// 	LastSegmentType LastSegmentType
// }

// func (n *segmentNode) findOrCreateChild(segment string) *segmentNode {
// 	if segment == "$" || (len(segment) > 0 && segment[0] == '$') {
// 		for _, child := range n.dynChildren {
// 			if child.paramName == segment[1:] || (segment == "$" && child.nodeType == nodeSplat) {
// 				return child
// 			}
// 		}
// 		return n.addDynamicChild(segment)
// 	}
// 	if n.children == nil {
// 		n.children = make(map[string]*segmentNode)
// 	}
// 	if child, exists := n.children[segment]; exists {
// 		return child
// 	}
// 	child := &segmentNode{nodeType: nodeStatic}
// 	n.children[segment] = child
// 	return child
// }

// func (r *Router) AddRoute(pattern string) {
// 	segments := ParseSegments(pattern)
// 	r.AddRouteWithSegments(segments, false)
// }

// func (n *segmentNode) addDynamicChild(segment string) *segmentNode {
// 	child := &segmentNode{}
// 	if segment == "$" {
// 		child.nodeType = nodeSplat
// 	} else {
// 		child.nodeType = nodeDynamic
// 		child.paramName = segment[1:]
// 	}
// 	n.dynChildren = append(n.dynChildren, child)
// 	return child
// }

// func NewRouter() *Router {
// 	return &Router{
// 		root: &segmentNode{
// 			children: make(map[string]*segmentNode),
// 		},
// 		staticRoutes: make(map[string][]staticRoute),
// 	}
// }

// func (r *Router) FindAllMatches(segments []string) ([]*Match, bool) {
// 	allMatches := collectMatches(r, segments)
// 	if len(allMatches) == 0 {
// 		return nil, false
// 	}
// 	sortMatchesAscending(allMatches)
// 	return allMatches, true
// }

// func (r *Router) FindBestMatch(req *http.Request) (*Match, bool) {
// 	segments := ParseSegments(req.URL.Path)
// 	allMatches := collectMatches(r, segments)
// 	if len(allMatches) == 0 {
// 		return nil, false
// 	}
// 	sortMatchesDescending(allMatches)
// 	return allMatches[0], true
// }

// func collectMatches(r *Router, segments []string) []*Match {
// 	path := "/" + strings.Join(segments, "/")
// 	var all []*Match

// 	if staticRoutes, ok := r.staticRoutes[path]; ok {
// 		for _, sr := range staticRoutes {
// 			all = append(all, &Match{
// 				Pattern:         path,
// 				Score:           sr.score,
// 				LastSegmentType: sr.lastSegmentType,
// 			})
// 		}
// 	}
// 	traverse, getMatches := makeTraverseFunc(segments)
// 	traverse(r.root, 0, 0)
// 	dynMatches := getMatches()
// 	all = append(all, dynMatches...)
// 	all = deduplicateMatches(all)
// 	return all
// }

// func deduplicateMatches(matches []*Match) []*Match {
// 	if len(matches) == 0 {
// 		return matches
// 	}
// 	type matchKey struct {
// 		pattern string
// 		segType LastSegmentType
// 	}
// 	unique := make(map[matchKey]*Match)
// 	for _, m := range matches {
// 		k := matchKey{m.Pattern, m.LastSegmentType}
// 		if old, ok := unique[k]; !ok || m.Score > old.Score {
// 			unique[k] = m
// 		}
// 	}
// 	result := make([]*Match, 0, len(unique))
// 	for _, m := range unique {
// 		result = append(result, m)
// 	}
// 	return result
// }

// func sortMatchesAscending(matches []*Match) {
// 	sort.Slice(matches, func(i, j int) bool {
// 		if matches[i].Score != matches[j].Score {
// 			return matches[i].Score < matches[j].Score
// 		}
// 		typeScore := map[LastSegmentType]int{
// 			LastSegmentTypes.Static:  4,
// 			LastSegmentTypes.Index:   3,
// 			LastSegmentTypes.Dynamic: 2,
// 			LastSegmentTypes.Splat:   1,
// 		}
// 		return typeScore[matches[i].LastSegmentType] > typeScore[matches[j].LastSegmentType]
// 	})
// }

// func sortMatchesDescending(matches []*Match) {
// 	sort.Slice(matches, func(i, j int) bool {
// 		if matches[i].Score != matches[j].Score {
// 			return matches[i].Score > matches[j].Score
// 		}
// 		typeScore := map[LastSegmentType]int{
// 			LastSegmentTypes.Static:  4,
// 			LastSegmentTypes.Index:   3,
// 			LastSegmentTypes.Dynamic: 2,
// 			LastSegmentTypes.Splat:   1,
// 		}
// 		return typeScore[matches[i].LastSegmentType] > typeScore[matches[j].LastSegmentType]
// 	})
// }

// func (r *Router) AddRouteWithSegments(segments []string, isIndex bool) {
// 	pattern := "/" + strings.Join(segments, "/")
// 	var totalScore int
// 	var lastSegType LastSegmentType

// 	if len(segments) == 0 {
// 		totalScore = scoreStatic
// 		lastSegType = LastSegmentTypes.Index
// 	} else {
// 		for _, segment := range segments {
// 			switch {
// 			case segment == "$":
// 				totalScore += scoreSplat
// 			case len(segment) > 0 && segment[0] == '$':
// 				totalScore += scoreDynamic
// 			default:
// 				totalScore += scoreStatic
// 			}
// 		}
// 		lastSegment := segments[len(segments)-1]
// 		switch {
// 		case lastSegment == "$":
// 			lastSegType = LastSegmentTypes.Splat
// 		case len(lastSegment) > 0 && lastSegment[0] == '$':
// 			lastSegType = LastSegmentTypes.Dynamic
// 		default:
// 			if isIndex {
// 				lastSegType = LastSegmentTypes.Index
// 			} else {
// 				lastSegType = LastSegmentTypes.Static
// 			}
// 		}
// 	}

// 	current := r.root
// 	for i, segment := range segments {
// 		child := current.findOrCreateChild(segment)
// 		if i == len(segments)-1 {
// 			child.routes = append(child.routes, routeInfo{
// 				Pattern:         pattern,
// 				Score:           totalScore,
// 				LastSegmentType: lastSegType,
// 			})
// 		}
// 		current = child
// 	}

// 	if lastSegType == LastSegmentTypes.Static || lastSegType == LastSegmentTypes.Index {
// 		r.staticRoutes[pattern] = append(r.staticRoutes[pattern], staticRoute{
// 			score:           totalScore,
// 			lastSegmentType: lastSegType,
// 		})
// 	}
// }

// type traverseFunc func(node *segmentNode, depth int, score int)

// func makeTraverseFunc(segments []string) (traverseFunc, func() []*Match) {
// 	var matches []*Match
// 	currentParams := make(Params)
// 	var splatSegments []string

// 	var traverse traverseFunc
// 	traverse = func(node *segmentNode, depth int, score int) {
// 		for _, rInfo := range node.routes {
// 			if rInfo.LastSegmentType == LastSegmentTypes.Index && depth < len(segments) {
// 				continue
// 			}

// 			paramsCopy := make(Params)
// 			for k, v := range currentParams {
// 				paramsCopy[k] = v
// 			}
// 			if len(paramsCopy) == 0 {
// 				paramsCopy = nil
// 			}

// 			var matchSplat []string
// 			if rInfo.LastSegmentType == LastSegmentTypes.Splat && depth < len(segments) {
// 				matchSplat = append([]string(nil), segments[depth:]...)
// 			} else if len(splatSegments) > 0 {
// 				matchSplat = append([]string(nil), splatSegments...)
// 			}

// 			matches = append(matches, &Match{
// 				Pattern:         rInfo.Pattern,
// 				Params:          paramsCopy,
// 				SplatSegments:   matchSplat,
// 				Score:           rInfo.Score,
// 				LastSegmentType: rInfo.LastSegmentType,
// 			})
// 		}

// 		if depth >= len(segments) {
// 			return
// 		}

// 		segment := segments[depth]
// 		childMatched := false

// 		if c, ok := node.children[segment]; ok {
// 			childMatched = true
// 			traverse(c, depth+1, score+scoreStatic)
// 		}

// 		for _, c := range node.dynChildren {
// 			if c.nodeType == nodeDynamic {
// 				childMatched = true
// 				currentParams[c.paramName] = segment
// 				traverse(c, depth+1, score+scoreDynamic)
// 				delete(currentParams, c.paramName)
// 			}
// 		}

// 		if !childMatched {
// 			for _, c := range node.dynChildren {
// 				if c.nodeType == nodeSplat {
// 					oldSplat := splatSegments
// 					splatSegments = segments[depth:] // capture all remaining
// 					traverse(c, len(segments), score+scoreSplat)
// 					splatSegments = oldSplat
// 				}
// 			}
// 		}
// 	}

// 	return traverse, func() []*Match { return matches }
// }

// func ParseSegments(path string) []string {
// 	if path == "" {
// 		return nil
// 	}

// 	// Estimate capacity
// 	maxSegments := 1
// 	for i := 0; i < len(path); i++ {
// 		if path[i] == '/' {
// 			maxSegments++
// 		}
// 	}

// 	segments := make([]string, 0, maxSegments)
// 	start := 0

// 	for i := 0; i <= len(path); i++ {
// 		if i == len(path) || path[i] == '/' {
// 			if i > start {
// 				segments = append(segments, path[start:i])
// 			}
// 			start = i + 1
// 		}
// 	}

// 	return segments
// }

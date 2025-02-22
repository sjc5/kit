package router

import (
	"net/http"
	"strings"
	"sync"
)

// LastSegmentType enumerates how the *last* segment of a route is interpreted.
type LastSegmentType string

var LastSegmentTypes = struct {
	Static  LastSegmentType
	Dynamic LastSegmentType
	Splat   LastSegmentType
	Index   LastSegmentType
}{
	Static:  "Static",
	Dynamic: "Dynamic",
	Splat:   "Splat",
	Index:   "Index",
}

// Params holds the name/value pairs for dynamic segments.
type Params map[string]string

// Sync pool for params to reduce allocation pressure
var paramsPool = sync.Pool{
	New: func() interface{} {
		return make(Params, 4)
	},
}

// Match is the result of a router match (for nested or single best match).
type Match struct {
	Pattern         string
	Params          Params
	SplatSegments   []string
	LastSegmentType LastSegmentType
}

// Route stores pattern metadata within trie nodes.
type Route struct {
	pattern         string
	paramNames      []string
	isIndex         bool
	lastSegmentType LastSegmentType
}

// Each node can have:
//
//   - staticChildren: exact matches for a segment
//   - dynamicChild: a single "$param" child
//   - splatChild:   a single "$" child
//
// We store up to two "nested style" routes (routeNonIndex + routeIndex) and
// a single best‐match route for the simpler scenario.
type node struct {
	staticChildren map[string]*node

	dynamicChild *node
	dynamicName  string

	splatChild *node

	// For nested usage:
	routeNonIndex *Route // e.g. layout
	routeIndex    *Route // e.g. index route

	// For simple usage:
	routeBestMatch *Route
}

// Router holds the root node.
type Router struct {
	root *node
}

// NewRouter constructs an empty Router.
func NewRouter() *Router {
	return &Router{
		root: &node{
			staticChildren: make(map[string]*node),
		},
	}
}

// ----------------------------------------------------------------------------
// AddRoute: simpler "best‐match" usage
// ----------------------------------------------------------------------------

// AddRoute is the simpler method for the "simple routing scenarios" tests.
func (r *Router) AddRoute(pattern string) {
	segments := ParseSegments(pattern)
	r.addBestMatchRoute(segments, pattern, false)
}

func (r *Router) addBestMatchRoute(segments []string, pattern string, isIndex bool) {
	lastType := determineLastSegmentType(segments, isIndex)

	curr := r.root
	var paramNames []string

	for _, seg := range segments {
		switch {
		case seg == "$":
			if curr.splatChild == nil {
				curr.splatChild = &node{staticChildren: make(map[string]*node)}
			}
			curr = curr.splatChild

		case strings.HasPrefix(seg, "$"):
			if curr.dynamicChild == nil {
				curr.dynamicChild = &node{staticChildren: make(map[string]*node)}
			}
			curr = curr.dynamicChild
			curr.dynamicName = seg[1:]
			paramNames = append(paramNames, seg[1:])

		default:
			child, ok := curr.staticChildren[seg]
			if !ok {
				child = &node{staticChildren: make(map[string]*node)}
				curr.staticChildren[seg] = child
			}
			curr = child
		}
	}

	curr.routeBestMatch = &Route{
		pattern:         pattern,
		paramNames:      paramNames,
		isIndex:         isIndex,
		lastSegmentType: lastType,
	}
}

// ----------------------------------------------------------------------------
// AddRouteWithSegments: used by the "nested routing" test
// ----------------------------------------------------------------------------

func (r *Router) AddRouteWithSegments(segments []string, isIndex bool) {
	pattern := "/" + strings.Join(segments, "/")
	// special case for root index
	if isIndex && len(segments) == 0 {
		pattern = "/"
	}

	lastType := LastSegmentTypes.Static
	if isIndex {
		lastType = LastSegmentTypes.Index
	} else if len(segments) > 0 {
		last := segments[len(segments)-1]
		if last == "$" {
			lastType = LastSegmentTypes.Splat
		} else if strings.HasPrefix(last, "$") {
			lastType = LastSegmentTypes.Dynamic
		}
	}

	curr := r.root
	var paramNames []string

	for _, seg := range segments {
		switch {
		case seg == "$":
			if curr.splatChild == nil {
				curr.splatChild = &node{staticChildren: make(map[string]*node)}
			}
			curr = curr.splatChild

		case strings.HasPrefix(seg, "$"):
			if curr.dynamicChild == nil {
				curr.dynamicChild = &node{staticChildren: make(map[string]*node)}
			}
			curr = curr.dynamicChild
			paramName := seg[1:]
			curr.dynamicName = paramName
			paramNames = append(paramNames, paramName)

		default:
			child, ok := curr.staticChildren[seg]
			if !ok {
				child = &node{staticChildren: make(map[string]*node)}
				curr.staticChildren[seg] = child
			}
			curr = child
		}
	}

	route := &Route{
		pattern:         pattern,
		paramNames:      paramNames,
		isIndex:         isIndex,
		lastSegmentType: lastType,
	}

	if isIndex {
		curr.routeIndex = route
	} else {
		curr.routeNonIndex = route
	}
}

// ----------------------------------------------------------------------------
// FindAllMatches: used by the "nested routing" test
// ----------------------------------------------------------------------------

// FindAllMatches: returns one chain of matches (0 or more), plus a boolean ok.
func (r *Router) FindAllMatches(segments []string) ([]*Match, bool) {
	params := paramsPool.Get().(Params)
	chain := collectChain(r.root, segments, params)
	ok := len(chain) > 0
	if !ok {
		// Return params to pool if not used
		for k := range params {
			delete(params, k)
		}
		paramsPool.Put(params)
		return nil, false
	}
	return chain, true
}

// collectChain does a depth‐first search that either returns a *single* chain
// of matches or nil if the path is a dead end.
func collectChain(n *node, segs []string, params Params) []*Match {
	// 1) Start this node's partial chain with its routeNonIndex (if any).
	var partial []*Match
	if n.routeNonIndex != nil {
		partial = append(partial, newMatch(n.routeNonIndex, params, nil))
	}

	// 2) If no leftover segments, we might also have a routeIndex here.
	if len(segs) == 0 {
		if n.routeIndex != nil {
			partial = append(partial, newMatch(n.routeIndex, params, nil))
		}
		return partial // partial could be empty or have 1 or 2 matches
	}

	// 3) If leftover segments remain, we try children in the order:
	//    static → dynamic → splat.
	seg := segs[0]
	tail := segs[1:]

	// (a) static child - avoid map lookup if empty
	if len(n.staticChildren) > 0 {
		if child, ok := n.staticChildren[seg]; ok {
			if childChain := collectChain(child, tail, params); childChain != nil {
				return append(partial, childChain...)
			}
		}
	}

	// (b) dynamic child
	if n.dynamicChild != nil {
		// Clone params - reuse from pool if possible
		newParams := paramsPool.Get().(Params)
		for k, v := range params {
			newParams[k] = v
		}
		newParams[n.dynamicChild.dynamicName] = seg

		if childChain := collectChain(n.dynamicChild, tail, newParams); childChain != nil {
			return append(partial, childChain...)
		}

		// Return params to pool if not used
		for k := range newParams {
			delete(newParams, k)
		}
		paramsPool.Put(newParams)
	}

	// (c) splat child
	if n.splatChild != nil {
		if splatChain := collectChainSplat(n.splatChild, segs, params); splatChain != nil {
			return append(partial, splatChain...)
		}
	}

	// 4) If none of the children succeeded, this is a dead end → no match
	return nil
}

// collectChainSplat attempts to match leftover with a splat route.
func collectChainSplat(n *node, leftover []string, params Params) []*Match {
	// If there's a non‐index splat route, we can consume everything
	if n.routeNonIndex != nil && n.routeNonIndex.lastSegmentType == LastSegmentTypes.Splat {
		chain := []*Match{
			newMatch(n.routeNonIndex, params, leftover),
		}
		return chain
	}

	// If leftover is empty, we might have an index splat route
	if len(leftover) == 0 && n.routeIndex != nil && n.routeIndex.lastSegmentType == LastSegmentTypes.Splat {
		return []*Match{
			newMatch(n.routeIndex, params, leftover),
		}
	}

	// No match
	return nil
}

// ----------------------------------------------------------------------------
// Simple "best match" usage
// ----------------------------------------------------------------------------

func (r *Router) FindBestMatch(req *http.Request) (*Match, bool) {
	segs := ParseSegments(req.URL.Path)
	params := paramsPool.Get().(Params)
	match := bestMatchDFS(r.root, segs, params)

	if match == nil {
		// Return params to pool if not used
		for k := range params {
			delete(params, k)
		}
		paramsPool.Put(params)
		return nil, false
	}

	return match, true
}

func bestMatchDFS(n *node, segs []string, params Params) *Match {
	if len(segs) == 0 {
		if n.routeBestMatch != nil {
			return newMatch(n.routeBestMatch, params, nil)
		}
		return nil
	}

	seg := segs[0]
	tail := segs[1:]

	// static - avoid unnecessary map lookups
	if len(n.staticChildren) > 0 {
		if child, ok := n.staticChildren[seg]; ok {
			if m := bestMatchDFS(child, tail, params); m != nil {
				return m
			}
		}
	}

	// dynamic
	if n.dynamicChild != nil {
		// Clone params - reuse from pool when possible
		p2 := paramsPool.Get().(Params)
		for k, v := range params {
			p2[k] = v
		}
		p2[n.dynamicChild.dynamicName] = seg

		if m := bestMatchDFS(n.dynamicChild, tail, p2); m != nil {
			return m
		}

		// Return params to pool if not used
		for k := range p2 {
			delete(p2, k)
		}
		paramsPool.Put(p2)
	}

	// splat
	if n.splatChild != nil && n.splatChild.routeBestMatch != nil &&
		n.splatChild.routeBestMatch.lastSegmentType == LastSegmentTypes.Splat {
		return newMatch(n.splatChild.routeBestMatch, params, segs)
	}

	return nil
}

// ----------------------------------------------------------------------------
// Utilities
// ----------------------------------------------------------------------------

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

func determineLastSegmentType(segments []string, isIndex bool) LastSegmentType {
	if isIndex {
		return LastSegmentTypes.Index
	}
	if len(segments) == 0 {
		return LastSegmentTypes.Static
	}
	last := segments[len(segments)-1]

	// Optimize the check by comparing first byte first
	if len(last) > 0 && last[0] == '$' {
		if len(last) == 1 {
			return LastSegmentTypes.Splat
		}
		return LastSegmentTypes.Dynamic
	}

	return LastSegmentTypes.Static
}

func newMatch(r *Route, params Params, splat []string) *Match {
	if params == nil {
		params = make(Params)
	}
	return &Match{
		Pattern:         r.pattern,
		Params:          params,
		SplatSegments:   splat,
		LastSegmentType: r.lastSegmentType,
	}
}

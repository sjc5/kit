package router

import (
	"net/http"
	"strings"
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
// We store up to two “nested style” routes (routeNonIndex + routeIndex) and
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
// AddRoute: simpler “best‐match” usage
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
// AddRouteWithSegments: used by the “nested routing” test
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
// FindAllMatches: used by the “nested routing” test
// ----------------------------------------------------------------------------

func (r *Router) FindAllMatches(segments []string) ([]*Match, bool) {
	matches := collectAllMatches(r.root, segments, make(Params))
	ok := len(matches) > 0
	return matches, ok
}

// collectAllMatches follows the test’s rule:
//   - Always add routeNonIndex at each node you pass.
//   - If leftover=0, add routeIndex if present, and **stop** (no deeper match).
//   - Otherwise, attempt static child; if that yields matches, stop.
//     If none, attempt dynamic; if that yields matches, stop.
//     If none, attempt splat.
func collectAllMatches(n *node, segs []string, params Params) []*Match {
	results := []*Match{}

	// 1) routeNonIndex always matches if present
	if n.routeNonIndex != nil {
		results = append(results, newMatch(n.routeNonIndex, params, nil))
	}

	// 2) If no more segments, add routeIndex if present and return
	if len(segs) == 0 {
		if n.routeIndex != nil {
			results = append(results, newMatch(n.routeIndex, params, nil))
		}
		return results
	}

	seg := segs[0]
	tail := segs[1:]

	// 3) Attempt static child
	if child, ok := n.staticChildren[seg]; ok {
		childMatches := collectAllMatches(child, tail, params)
		if len(childMatches) > 0 {
			return append(results, childMatches...)
		}
	}

	// 4) Attempt dynamic child
	if n.dynamicChild != nil {
		newParams := cloneParams(params)
		newParams[n.dynamicChild.dynamicName] = seg
		childMatches := collectAllMatches(n.dynamicChild, tail, newParams)
		if len(childMatches) > 0 {
			return append(results, childMatches...)
		}
	}

	// 5) Attempt splat
	if n.splatChild != nil {
		splatMatches := collectAllMatchesSplat(n.splatChild, segs, params)
		if len(splatMatches) > 0 {
			return append(results, splatMatches...)
		}
	}

	return results
}

func collectAllMatchesSplat(n *node, leftover []string, params Params) []*Match {
	results := []*Match{}

	// The node can have a routeNonIndex with lastSegmentType == Splat
	if n.routeNonIndex != nil && n.routeNonIndex.lastSegmentType == LastSegmentTypes.Splat {
		results = append(results, newMatch(n.routeNonIndex, params, leftover))
	}

	// If leftover is empty and there's an index route that's Splat, match that
	if len(leftover) == 0 && n.routeIndex != nil && n.routeIndex.lastSegmentType == LastSegmentTypes.Splat {
		results = append(results, newMatch(n.routeIndex, params, leftover))
	}

	return results
}

// ----------------------------------------------------------------------------
// Simple “best match” usage
// ----------------------------------------------------------------------------

func (r *Router) FindBestMatch(req *http.Request) (*Match, bool) {
	segs := ParseSegments(req.URL.Path)
	params := make(Params)
	match := bestMatchDFS(r.root, segs, params)
	return match, (match != nil)
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

	// static
	if child, ok := n.staticChildren[seg]; ok {
		if m := bestMatchDFS(child, tail, params); m != nil {
			return m
		}
	}

	// dynamic
	if n.dynamicChild != nil {
		p2 := cloneParams(params)
		p2[n.dynamicChild.dynamicName] = seg
		if m := bestMatchDFS(n.dynamicChild, tail, p2); m != nil {
			return m
		}
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
	// The tests want an empty slice both for "" and "/"
	if path == "" || path == "/" {
		return []string{}
	}
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return []string{}
	}
	parts := strings.Split(path, "/")
	// Filter out any empty splits (in case of multiple //)
	var segs []string
	for _, p := range parts {
		if p != "" {
			segs = append(segs, p)
		}
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
	switch {
	case last == "$":
		return LastSegmentTypes.Splat
	case strings.HasPrefix(last, "$"):
		return LastSegmentTypes.Dynamic
	default:
		return LastSegmentTypes.Static
	}
}

func cloneParams(src Params) Params {
	dst := make(Params, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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

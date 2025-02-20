package router

import (
	"net/http"
)

// Match represents a route match result
type Match struct {
	Pattern       string
	Params        Params
	SplatSegments []string
	Score         int
}

const (
	nodeStatic  uint8 = 0
	nodeDynamic uint8 = 1
	nodeSplat   uint8 = 2

	// Scoring weights for route specificity
	scoreStatic  = 3
	scoreDynamic = 2
	scoreSplat   = 1
)

// segmentNode represents a node in the routing trie
type segmentNode struct {
	pattern     string
	nodeType    uint8
	children    map[string]*segmentNode
	dynChildren []*segmentNode
	paramName   string
	finalScore  int
}

// Router manages route matching
type Router struct {
	root         *segmentNode
	staticRoutes map[string]int
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		root:         &segmentNode{},
		staticRoutes: make(map[string]int),
	}
}

// AddRoute registers a new route pattern
func (r *Router) AddRoute(pattern string) {
	segments := ParseSegments(pattern)

	// Calculate score and check if static in one pass
	totalScore := 0
	isStatic := true
	for _, segment := range segments {
		switch {
		case segment == "$":
			totalScore += scoreSplat
			isStatic = false
		case len(segment) > 0 && segment[0] == '$':
			totalScore += scoreDynamic
			isStatic = false
		default:
			totalScore += scoreStatic
		}
	}

	if isStatic {
		r.staticRoutes[pattern] = totalScore
		return
	}

	current := r.root
	nodeScore := 0

	for i, segment := range segments {
		child := current.findOrCreateChild(segment)
		switch {
		case segment == "$":
			nodeScore += scoreSplat
		case len(segment) > 0 && segment[0] == '$':
			nodeScore += scoreDynamic
		default:
			nodeScore += scoreStatic
		}

		if i == len(segments)-1 {
			child.finalScore = nodeScore
			child.pattern = pattern
		}
		current = child
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

// FindBestMatch finds the best matching route for an HTTP request
func (r *Router) FindBestMatch(req *http.Request) (*Match, bool) {
	path := req.URL.Path
	if score, ok := r.staticRoutes[path]; ok {
		return &Match{Pattern: path, Score: score}, true
	}

	segments := ParseSegments(path)
	return r.findBestMatchInner(segments)
}

// findBestMatchInner performs the recursive route matching
func (r *Router) findBestMatchInner(segments []string) (*Match, bool) {
	var bestMatch *Match
	currentParams := make(Params)

	var traverse func(node *segmentNode, depth int, score int)
	traverse = func(node *segmentNode, depth int, score int) {
		if depth == len(segments) || node.nodeType == nodeSplat {
			if node.pattern != "" && (bestMatch == nil || score > bestMatch.Score) {
				bestMatch = &Match{
					Pattern: node.pattern,
					Score:   score,
				}
				if len(currentParams) > 0 {
					bestMatch.Params = make(Params, len(currentParams))
					for k, v := range currentParams {
						bestMatch.Params[k] = v
					}
				}
				if node.nodeType == nodeSplat && depth < len(segments) {
					bestMatch.SplatSegments = segments[depth:]
				}
			}
			return
		}

		segment := segments[depth]
		if node.children != nil {
			if child, ok := node.children[segment]; ok {
				traverse(child, depth+1, score+scoreStatic)
			}
		}

		for _, child := range node.dynChildren {
			switch child.nodeType {
			case nodeDynamic:
				currentParams[child.paramName] = segment
				traverse(child, depth+1, score+scoreDynamic)
				delete(currentParams, child.paramName)
			case nodeSplat:
				traverse(child, len(segments), score+scoreSplat)
			}
		}
	}

	traverse(r.root, 0, 0)
	return bestMatch, bestMatch != nil
}

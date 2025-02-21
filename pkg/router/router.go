package router

import "net/http"

const (
	nodeStatic  uint8 = 0
	nodeDynamic uint8 = 1
	nodeSplat   uint8 = 2

	scoreStatic  = 3
	scoreDynamic = 2
	scoreSplat   = 1
)

type segmentNode struct {
	pattern     string
	nodeType    uint8
	children    map[string]*segmentNode
	dynChildren []*segmentNode
	paramName   string
	finalScore  int
}

type Router struct {
	root         *segmentNode
	staticRoutes map[string]int
}

func NewRouter() *Router {
	return &Router{
		root:         &segmentNode{},
		staticRoutes: make(map[string]int),
	}
}

func (r *Router) AddRoute(pattern string) {
	segments := ParseSegments(pattern)
	r.AddRouteWithSegments(pattern, segments)
}

func (r *Router) AddRouteWithSegments(pattern string, segments []string) {
	if len(segments) > 0 {
		if segments[len(segments)-1] == "_index" {
			segments = segments[:len(segments)-1]
		}
	}

	var totalScore int
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
	var nodeScore int

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

type Params = map[string]string

type Match struct {
	Pattern       string
	Params        Params
	SplatSegments []string
	Score         int
}

func (r *Router) FindBestMatch(req *http.Request) (*Match, bool) {
	path := req.URL.Path
	if score, ok := r.staticRoutes[path]; ok {
		return &Match{Pattern: path, Score: score}, true
	}

	segments := ParseSegments(path)
	return r.findBestMatchInner(segments)
}

func (r *Router) findBestMatchInner(segments []string) (*Match, bool) {
	traverse, getBestMatch, _ := makeTraverseFunc(segments, false)
	traverse(r.root, 0, 0)
	bestMatch := getBestMatch()
	return bestMatch, bestMatch != nil
}

func (r *Router) FindAllMatches(segments []string) ([]*Match, bool) {
	traverse, _, getAllMatches := makeTraverseFunc(segments, true)
	traverse(r.root, 0, 0)
	allMatches := getAllMatches()
	if len(allMatches) == 0 {
		return nil, false
	}
	return allMatches, true
}

type traverseFunc func(node *segmentNode, depth int, score int)

func makeTraverseFunc(segments []string, findAll bool) (traverseFunc, func() *Match, func() []*Match) {
	var bestMatch *Match
	var allMatches []*Match
	currentParams := make(Params)

	var traverse traverseFunc
	traverse = func(node *segmentNode, depth int, score int) {
		if depth == len(segments) || node.nodeType == nodeSplat {
			if node.pattern != "" {
				// Avoid unnecessary allocations
				var paramsCopy Params
				if len(currentParams) > 0 {
					paramsCopy = make(Params, len(currentParams))
					for k, v := range currentParams {
						paramsCopy[k] = v
					}
				}

				// Lazy splat slicing to avoid unnecessary allocations
				var splatSegments []string
				if node.nodeType == nodeSplat && depth < len(segments) {
					splatSegments = segments[depth:]
				}

				match := &Match{
					Pattern:       node.pattern,
					Score:         score,
					Params:        paramsCopy,
					SplatSegments: splatSegments,
				}

				// Handle best match logic
				if !findAll {
					if bestMatch == nil || score > bestMatch.Score {
						bestMatch = match
					}
				} else {
					allMatches = append(allMatches, match)
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
				// Only set if needed to avoid unnecessary writes
				currentParams[child.paramName] = segment
				traverse(child, depth+1, score+scoreDynamic)
				delete(currentParams, child.paramName) // Minimize reallocation pressure
			case nodeSplat:
				traverse(child, len(segments), score+scoreSplat)
			}
		}
	}

	return traverse, func() *Match { return bestMatch }, func() []*Match { return allMatches }
}

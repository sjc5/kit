package router

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

func (router *RouterBest) makeTraverseFunc(segments []string, findBestOnly bool) (traverseFunc, func() []*Match) {
	var matches []*Match

	currentParams := make(Params)
	currentSplat := make([]string, 0) // Track splat segments during traversal

	var traverse traverseFunc
	traverse = func(node *segmentNode, depth int, score int) {
		// Reset splat segments at each new node traversal
		if len(currentSplat) > 0 {
			currentSplat = currentSplat[:0]
		}

		// If we're at the end or hit a splat, check for a match
		if depth == len(segments) || node.nodeType == nodeSplat {
			// Capture splat segments if we're at a splat node
			var splatValues []string
			if node.nodeType == nodeSplat && depth < len(segments) {
				// Efficiently append remaining segments
				splatValues = make([]string, 0, len(segments)-depth)
				splatValues = append(splatValues, segments[depth:]...)
			}

			// Copy params only if needed
			var paramsCopy Params
			if len(currentParams) > 0 {
				paramsCopy = make(Params, len(currentParams))
				for k, v := range currentParams {
					paramsCopy[k] = v
				}
			}

			match := &Match{
				Params:      paramsCopy,
				SplatValues: splatValues,
				Score:       score,
			}

			if findBestOnly {
				if matches == nil {
					matches = append(matches, match)
				}
				if len(matches) == 0 || matches[0] == nil || score > matches[0].Score {
					matches[0] = match
				}
			} else {
				matches = append(matches, match)
			}
			return
		}

		if depth >= len(segments) {
			return
		}

		segment := segments[depth]
		if node.children != nil {
			if child, ok := node.children[segment]; ok {
				traverse(child, depth+1, score+scoreStaticMatch)
			}
		}

		for _, child := range node.dynChildren {
			switch child.nodeType {
			case nodeDynamic:
				currentParams[child.paramName] = segment
				traverse(child, depth+1, score+scoreDynamic)
				delete(currentParams, child.paramName)
			case nodeSplat:
				// Traverse with splat, maintaining full score
				traverse(child, depth, score+scoreSplat)
			}
		}
	}

	return traverse, func() []*Match { return matches }
}

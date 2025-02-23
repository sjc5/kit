package router

import (
	"fmt"
	"sort"
	"strings"
)

func (r *RouterBest) PrintReadableTrie() {
	fmt.Println("Routing Trie Structure:")
	fmt.Println("======================")
	if r.trie.root == nil {
		fmt.Println("Empty trie")
		return
	}

	// Print static routes (sorted)
	if len(r.trie.staticRoutes) > 0 {
		fmt.Println("Static Routes:")
		// Collect and sort static routes
		staticPatterns := make([]string, 0, len(r.trie.staticRoutes))
		for pattern := range r.trie.staticRoutes {
			staticPatterns = append(staticPatterns, pattern)
		}
		sort.Strings(staticPatterns)

		for _, pattern := range staticPatterns {
			fmt.Printf("  %s (score: %d)\n", pattern, r.trie.staticRoutes[pattern])
		}
		fmt.Println()
	}

	// Print dynamic trie
	fmt.Println("Dynamic Trie:")
	printNode(r.trie.root, 0, "")
	fmt.Println("======================")
}

func printNode(node *segmentNode, depth int, prefix string) {
	indent := strings.Repeat("  ", depth)

	// Print current node info if it has a pattern
	if node.pattern != "" {
		nodeTypeStr := "static"
		if node.nodeType == nodeDynamic {
			nodeTypeStr = "dynamic"
		} else if node.nodeType == nodeSplat {
			nodeTypeStr = "splat"
		}

		fmt.Printf("%s%s (type: %s, score: %d)\n",
			indent,
			node.pattern,
			nodeTypeStr,
			node.finalScore,
		)
	}

	// Print static children (sorted)
	if len(node.children) > 0 {
		staticSegments := make([]string, 0, len(node.children))
		for segment := range node.children {
			staticSegments = append(staticSegments, segment)
		}
		sort.Strings(staticSegments)

		for _, segment := range staticSegments {
			fmt.Printf("%s%s/\n", indent, segment)
			printNode(node.children[segment], depth+1, prefix+segment+"/")
		}
	}

	// Print dynamic children (sorted)
	if len(node.dynChildren) > 0 {
		// Sort dynamic children by paramName (for dynamic) and nodeType
		sortedDynChildren := make([]*segmentNode, len(node.dynChildren))
		copy(sortedDynChildren, node.dynChildren)
		sort.Slice(sortedDynChildren, func(i, j int) bool {
			// Splat ($) comes last
			if sortedDynChildren[i].nodeType == nodeSplat && sortedDynChildren[j].nodeType != nodeSplat {
				return false
			}
			if sortedDynChildren[i].nodeType != nodeSplat && sortedDynChildren[j].nodeType == nodeSplat {
				return true
			}
			// For dynamic nodes, sort by paramName
			return sortedDynChildren[i].paramName < sortedDynChildren[j].paramName
		})

		for _, child := range sortedDynChildren {
			if child.nodeType == nodeDynamic {
				fmt.Printf("%s$%s/\n", indent, child.paramName)
			} else if child.nodeType == nodeSplat {
				fmt.Printf("%s$/\n", indent)
			}
			printNode(child, depth+1, prefix)
		}
	}
}

func (router *RouterBest) PrintRouteMaps() {
	fmt.Println("******************")
	fmt.Println("STATIC ROUTES:")
	for k := range router.StaticRegisteredRoutes {
		fmt.Println(k)
	}
	fmt.Println()
	fmt.Println("DYNAMIC ROUTES:")
	for k := range router.DynamicRegisteredRoutes {
		fmt.Println(k)
	}
	fmt.Println()
}

package matcher

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

package router

func ParseSegments(path string) []string {
	if path == "" {
		return nil
	}

	// Estimate capacity
	maxSegments := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			maxSegments++
		}
	}

	segments := make([]string, 0, maxSegments)
	start := 0

	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
		}
	}

	return segments
}

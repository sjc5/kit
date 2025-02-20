package router

type Params = map[string]string

type Results struct {
	Params             Params
	Score              int
	RealSegmentsLength int
	SplatSegments      []string
}

func MatchCore(patternSegments []string, realSegments []string) (*Results, bool) {
	if len(patternSegments) > len(realSegments) {
		return nil, false
	}

	realSegmentsLength := len(realSegments)
	params := make(Params)
	score := 0
	var splatSegments []string

	for i, ps := range patternSegments {
		if i >= len(realSegments) {
			return nil, false
		}

		isLastSegment := i == len(patternSegments)-1

		switch {
		case ps == realSegments[i]:
			score += 3 // Exact match
		case ps == "$":
			score += 1 // Splat segment
			if isLastSegment {
				splatSegments = realSegments[i:]
			}
		case len(ps) > 0 && ps[0] == '$':
			score += 2 // Dynamic parameter
			params[ps[1:]] = realSegments[i]
		default:
			return nil, false
		}
	}

	results := &Results{
		Params:             params,
		Score:              score,
		RealSegmentsLength: realSegmentsLength,
	}

	if splatSegments != nil {
		results.SplatSegments = splatSegments
	}

	return results, true
}

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

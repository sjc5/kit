package matcher

import (
	"strings"

	"github.com/sjc5/kit/pkg/router"
)

func PatternToRegisteredPath(pattern string) *RegisteredPath {
	rawSegments := router.ParseSegments(pattern)

	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		// Skip double underscore segments
		if len(segment) > 1 && segment[0] == '_' && segment[1] == '_' {
			continue
		}

		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		panic("This shouldn't happen. Make sure your root index pattern is \"/_index\"")
	}

	lastSegment := segments[len(segments)-1]

	var routeType PathType

	switch {
	case lastSegment == "_index":
		routeType = PathTypes.Index
	case len(segments) == 1 && lastSegment == "$":
		routeType = PathTypes.UltimateCatch
	case lastSegment == "$":
		routeType = PathTypes.NonUltimateSplat
	case len(lastSegment) > 1 && lastSegment[0] == '$':
		routeType = PathTypes.DynamicLayout
	default:
		routeType = PathTypes.StaticLayout
	}

	return &RegisteredPath{
		Pattern:  "/" + strings.Join(segments, "/"),
		Segments: segments,
		PathType: routeType,
	}
}

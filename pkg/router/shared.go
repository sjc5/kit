package router

const (
	scoreStaticMatch = 3
	scoreDynamic     = 2
	scoreSplat       = 1
)

type Params = map[string]string

type Match struct {
	*RegisteredRoute

	Params      Params
	SplatValues []string
	Score       int

	notes string
}

type SegmentType = string

var SegmentTypes = struct {
	Splat   SegmentType
	Static  SegmentType
	Dynamic SegmentType
	Index   SegmentType
}{
	Splat:   "splat",
	Static:  "static",
	Dynamic: "dynamic",
	Index:   "index",
}

type Segment struct {
	Value string
	Type  string
}

type RegisteredRoute struct {
	Pattern  string
	Segments []*Segment
}

func (rr *RegisteredRoute) GetLastSegmentType() SegmentType {
	return rr.Segments[len(rr.Segments)-1].Type
}

func (rr *RegisteredRoute) LastSegmentIsIndex() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Index
}

func (rr *RegisteredRoute) LastSegmentIsSplat() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Splat
}

func (rr *RegisteredRoute) LastSegmentIsUltimateCatch() bool {
	return rr.LastSegmentIsSplat() && len(rr.Segments) == 1
}

func (rr *RegisteredRoute) LastSegmentIsStaticLayout() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Static
}

func (rr *RegisteredRoute) LastSegmentIsNonUltimateSplat() bool {
	return rr.LastSegmentIsSplat() && len(rr.Segments) > 1
}

func (rr *RegisteredRoute) LastSegmentIsDynamicLayout() bool {
	return rr.GetLastSegmentType() == SegmentTypes.Dynamic
}

func (rr *RegisteredRoute) LastSegmentIsIndexPrecededByDynamic() bool {
	segmentsLen := len(rr.Segments)

	return rr.LastSegmentIsIndex() &&
		segmentsLen >= 2 &&
		rr.Segments[segmentsLen-2].Type == SegmentTypes.Dynamic
}

func (rr *RegisteredRoute) IndexAdjustedPatternLen() int {
	if rr.LastSegmentIsIndex() {
		return len(rr.Segments) - 1
	}
	return len(rr.Segments)
}

type Pattern = string

type RouterBest struct {
	NestedIndexSignifier string
	// e.g., "_index"

	ShouldExcludeSegmentFunc func(segment string) bool
	// e.g., return strings.HasPrefix(segment, "__")

	trie *trie

	StaticRegisteredRoutes  map[Pattern]*RegisteredRoute
	DynamicRegisteredRoutes map[Pattern]*RegisteredRoute
}

package router

import "fmt"

const (
	defaultNestedIndexSignifier = "_index"
	defaultDynamicParamPrefix   = ':'
	defaultSplatSegmentRune     = '*'

	nodeStatic  uint8 = 0
	nodeDynamic uint8 = 1
	nodeSplat   uint8 = 2

	scoreStaticMatch = 3
	scoreDynamic     = 2
	scoreSplat       = 1
)

type Params = map[string]string

type pattern = string
type segType = string
type patternsMap = map[pattern]*RegisteredPattern
type matchesMap = map[pattern]*Match

type Match struct {
	*RegisteredPattern
	Params      Params
	SplatValues []string

	score uint16
}

type Matcher struct {
	middlewares []Middleware

	staticPatterns  patternsMap
	dynamicPatterns patternsMap
	rootNode        *segmentNode

	// options
	nestedIndexSignifier string
	// __TODO just scrap this, consumer should handle first before registration
	shouldExcludeSegmentFunc func(segment string) bool
	dynamicParamPrefixRune   rune
	splatSegmentRune         rune

	// pre-computed values
	catchAllPattern           string
	slashNestedIndexSignifier string
}

func (m *Matcher) AddMiddleware(middleware Middleware) *Matcher {
	m.middlewares = append(m.middlewares, middleware)
	return m
}

func (m *Matcher) AddMiddlewareToPattern(pattern string, middleware Middleware) *Matcher {
	if rp, ok := m.staticPatterns[pattern]; ok {
		rp.AddMiddleware(middleware)
	} else if rp, ok := m.dynamicPatterns[pattern]; ok {
		rp.AddMiddleware(middleware)
	} else {
		panic(fmt.Sprintf("pattern %s not found", pattern))
	}
	return m
}

type segment struct {
	value   string
	segType segType
}

var segTypes = struct {
	splat   segType
	static  segType
	dynamic segType
	index   segType
}{
	splat:   "splat",
	static:  "static",
	dynamic: "dynamic",
	index:   "index",
}

type MatcherOptions struct {
	// Required for nested matcher, not required for non-nested. Defaults to "_index".
	NestedIndexSignifier string

	// Optional. Defaults to ':'.
	DynamicParamPrefixRune rune

	// Optional. Defaults to '*'.
	SplatSegmentRune rune

	// Optional. e.g., return strings.HasPrefix(segment, "__")
	// useful if you're using a file system as the source for your patterns and want to "skip" certain directories
	ShouldExcludeSegmentFunc func(segment string) bool
}

func NewMatcher(options *MatcherOptions) *Matcher {
	var instance = new(Matcher)

	instance.rootNode = new(segmentNode)
	instance.staticPatterns = make(patternsMap)
	instance.dynamicPatterns = make(patternsMap)

	if options != nil {
		if options.NestedIndexSignifier == "" {
			instance.nestedIndexSignifier = defaultNestedIndexSignifier
		} else {
			instance.nestedIndexSignifier = options.NestedIndexSignifier
		}

		if options.DynamicParamPrefixRune == 0 {
			instance.dynamicParamPrefixRune = defaultDynamicParamPrefix
		} else {
			instance.dynamicParamPrefixRune = options.DynamicParamPrefixRune
		}

		if options.SplatSegmentRune == 0 {
			instance.splatSegmentRune = defaultSplatSegmentRune
		} else {
			instance.splatSegmentRune = options.SplatSegmentRune
		}

		instance.shouldExcludeSegmentFunc = options.ShouldExcludeSegmentFunc
	} else {
		instance.nestedIndexSignifier = defaultNestedIndexSignifier
		instance.dynamicParamPrefixRune = defaultDynamicParamPrefix
		instance.splatSegmentRune = defaultSplatSegmentRune
	}

	instance.catchAllPattern = "/" + string(instance.splatSegmentRune)
	instance.slashNestedIndexSignifier = "/" + instance.nestedIndexSignifier

	return instance
}

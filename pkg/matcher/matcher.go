package matcher

import "github.com/sjc5/kit/pkg/opt"

type (
	Params = map[string]string

	pattern     = string
	segType     = string
	patternsMap = map[pattern]*RegisteredPattern
	matchesMap  = map[pattern]*Match
)

type Matcher struct {
	staticPatterns  patternsMap
	dynamicPatterns patternsMap
	rootNode        *segmentNode

	nestedIndexSignifier   string
	dynamicParamPrefixRune rune
	splatSegmentRune       rune

	catchAllPattern           string
	slashNestedIndexSignifier string
}

type Match struct {
	*RegisteredPattern
	Params      Params
	SplatValues []string

	score uint16
}

type Options struct {
	DynamicParamPrefixRune rune   // Optional. Defaults to ':'.
	SplatSegmentRune       rune   // Optional. Defaults to '*'.
	NestedIndexSignifier   string // Required for nested matcher, not required for non-nested. Defaults to "_index".
}

func New(opts *Options) *Matcher {
	var instance = new(Matcher)

	instance.staticPatterns = make(patternsMap)
	instance.dynamicPatterns = make(patternsMap)
	instance.rootNode = new(segmentNode)

	if opts == nil {
		opts = new(Options)
	}
	instance.nestedIndexSignifier = opt.Resolve(opts, opts.NestedIndexSignifier, "_index")
	instance.dynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	instance.splatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	instance.catchAllPattern = "/" + string(instance.splatSegmentRune)
	instance.slashNestedIndexSignifier = "/" + instance.nestedIndexSignifier

	return instance
}

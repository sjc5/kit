package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/opt"
	"github.com/sjc5/kit/pkg/tasks"
)

type Router struct {
	marshalInput       func(r *http.Request, iPtr any) error
	tasksRegistry      *tasks.Registry
	classicMiddlewares []ClassicMiddleware
	taskMiddlewares    []tasks.AnyTask
	methodToMatcherMap map[method]*decoratedMatcher
	matcherOptions     *matcher.Options
	notFoundHandler    ClassicHandler
}

func newRouter(opts *Options) *Router {
	matcherOptions := new(matcher.Options)

	if opts == nil {
		opts = new(Options)
	}
	matcherOptions.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	matcherOptions.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	return &Router{
		marshalInput:       opts.MarshalInput,
		tasksRegistry:      opts.TasksRegistry,
		methodToMatcherMap: make(map[method]*decoratedMatcher),
		matcherOptions:     matcherOptions,
	}
}

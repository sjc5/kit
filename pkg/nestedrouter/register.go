package nestedrouter

import (
	"fmt"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

var API_SEGMENT = "api" // __TODO Make this settable in config

type RegisteredPattern struct {
	matcherRP *matcher.RegisteredPattern
}

func (router *Router) RegisterPattern(pattern string) *RegisteredPattern {
	return &RegisteredPattern{matcherRP: router.matcher.RegisterPattern(pattern)}
}

func RegisterPatternWithLoader[O any](router *Router, pattern string, loader Loader[O]) *Router {
	router.RegisterPattern(pattern)

	taskFunc := func(_ *tasks.Ctx, routerCtx *Ctx) (O, error) {
		return loader(routerCtx)
	}

	router.loaders[pattern] = tasks.New(router.tasksRegistry, taskFunc)

	return router
}

type actionCtxWrapper[I any] struct {
	routerCtx *Ctx
	input     I
}

func RegisterPatternWithQuery[I any, O any](router *Router, pattern string, query Action[I, O]) *Router {
	pattern = fmt.Sprintf("/%s%s", API_SEGMENT, pattern)

	router.RegisterPattern(pattern)

	taskFunc := func(_ *tasks.Ctx, actionCtxWrapper *actionCtxWrapper[I]) (O, error) {
		return query(actionCtxWrapper.routerCtx, actionCtxWrapper.input)
	}

	router.queries[pattern] = tasks.New(router.tasksRegistry, taskFunc)

	return router
}

/*
4 PATHS:
HTML UI
JSON UI
JSON QUERY
JSON MUTATION
*/

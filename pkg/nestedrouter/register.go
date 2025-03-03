package nestedrouter

import (
	"fmt"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

type CtxInput[I any] struct {
	Input I
	*Ctx
}

var API_SEGMENT = "api" // __TODO Make this settable in config

type RegisteredPattern struct {
	matcherRP *matcher.RegisteredPattern
}

func (router *Router) RegisterPattern(pattern string) *RegisteredPattern {
	return &RegisteredPattern{matcherRP: router.matcher.RegisterPattern(pattern)}
}

func RegisterPatternWithLoader[I any, O any](router *Router, pattern string, loader tasks.TaskFn[*Ctx, O]) *Router {
	router.RegisterPattern(pattern)

	router.loaders[pattern] = tasks.New(router.tasksRegistry, func(c *tasks.TasksCtxWithInput[*Ctx]) (O, error) {
		return loader(c)
	})

	return router
}

func RegisterPatternWithQuery[I any, O any](router *Router, pattern string, query tasks.TaskFn[*CtxInput[I], O]) *Router {
	pattern = fmt.Sprintf("/%s%s", API_SEGMENT, pattern)

	router.RegisterPattern(pattern)

	router.queries[pattern] = tasks.New(router.tasksRegistry, func(c *tasks.TasksCtxWithInput[*CtxInput[I]]) (O, error) {
		return query(c)
	})

	return router
}

/*
4 PATHS:
HTML UI
JSON UI
JSON QUERY
JSON MUTATION
*/

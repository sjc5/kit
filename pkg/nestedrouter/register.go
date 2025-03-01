package nestedrouter

import (
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

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

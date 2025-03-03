package router

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

func mustRegisterPattern[I any, O any](rp *RegisteredPattern[I, O]) {
	m := mustGetMatcher(rp.router, rp.method)

	m.matcher.RegisterPattern(rp.pattern)

	m.registeredPatterns[rp.pattern] = rp

	if rp.handlerType == handlerTypes.task {
		m.ctxGetters[rp.pattern] = toCtxGetterImpl[I](rp)
	}
}

func toCtxGetterImpl[I any, O any](rp *RegisteredPattern[I, O]) ctxGetterImpl[I] {
	return ctxGetterImpl[I](
		func(r *http.Request, match *matcher.Match) *RouterCtx[I] {
			c := new(RouterCtx[I])
			if len(match.Params) > 0 {
				c.params = match.Params
			}
			if len(match.SplatValues) > 0 {
				c.splatValues = match.SplatValues
			}
			c.tasksCtx = rp.router.tasksRegistry.NewCtxFromRequest(r)

			iPtr := rp.Phantom().NewIPtr()
			if err := rp.router.marshalInput(c.Request(), iPtr); err != nil {
				// __TODO do something here
				fmt.Println("validation err", err)
			}
			c.input = *(iPtr.(*I))
			return c
		},
	)
}

type decoratedMatcher struct {
	matcher            *matcher.Matcher
	classicMiddlewares []ClassicMiddleware
	taskMiddlewares    []tasks.AnyTask
	registeredPatterns map[pattern]AnyRegisteredPattern
	ctxGetters         map[pattern]ctxGetter
}

func mustGetMatcher(router *Router, method method) *decoratedMatcher {
	m, err := getMatcher(router, method)
	if err != nil {
		panic(err)
	}
	return m
}

func getMatcher(router *Router, method method) (*decoratedMatcher, error) {
	if _, ok := permittedHTTPMethods[method]; !ok {
		return nil, errors.New("unknown method")
	}

	m, ok := router.methodToMatcherMap[method]
	if !ok {
		m = &decoratedMatcher{
			matcher:            matcher.New(router.matcherOptions),
			registeredPatterns: make(map[string]AnyRegisteredPattern),
			ctxGetters:         make(map[string]ctxGetter),
		}
		router.methodToMatcherMap[method] = m
	}

	return m, nil
}

package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/tasks"
)

type Options struct {
	MarshalInput           func(r *http.Request, iPtr any) error
	TasksRegistry          *tasks.Registry
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.
}

var NewRouter = newRouter

func FnToTaskHandler[I any, O any](router *Router, f TaskHandlerFn[I, O]) *TaskHandler[I, O] {
	return tasks.New(
		router.tasksRegistry,
		func(c *tasks.TasksCtxWithInput[TaskHandlerI[I]]) (O, error) {
			return f(c.Input)
		},
	)
}

func FnToTaskMiddleware[O any](router *Router, f TaskMiddlewareFn[O]) *TaskMiddleware[O] {
	return tasks.New(
		router.tasksRegistry,
		func(c *tasks.TasksCtxWithInput[TaskMiddlewareI]) (O, error) {
			return f(c.Input)
		},
	)
}

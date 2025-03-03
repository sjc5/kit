package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/datafn"
	"github.com/sjc5/kit/pkg/tasks"
)

type TaskMiddlewareI = *http.Request
type TaskMiddlewareFn[O any] = datafn.Fn[TaskMiddlewareI, O]
type TaskMiddleware[O any] = tasks.Task[TaskMiddlewareI, O]

type TaskHandlerI[I any] = *RouterCtx[I]
type TaskHandlerFn[I any, O any] = datafn.Fn[TaskHandlerI[I], O]
type TaskHandler[I any, O any] = tasks.Task[TaskHandlerI[I], O]

type None = struct{}

/////////////////////////////////////////////////////////////////////
/////// NOT FOUND HANDLERS
/////////////////////////////////////////////////////////////////////

func NotFoundHandler(router *Router, h ClassicHandler) {
	router.notFoundHandler = h
}

/////////////////////////////////////////////////////////////////////
/////// GLOBAL MIDDLEWARES
/////////////////////////////////////////////////////////////////////

func GlobalMiddlewareClassic(router *Router, mw ClassicMiddleware) {
	router.classicMiddlewares = append(router.classicMiddlewares, mw)
}

func GlobalMiddleware[O any](router *Router, task *TaskMiddleware[O]) None {
	router.taskMiddlewares = append(router.taskMiddlewares, task)
	return None{}
}

/////////////////////////////////////////////////////////////////////
/////// METHOD MIDDLEWARES
/////////////////////////////////////////////////////////////////////

func MethodMiddlewareClassic(router *Router, method method, mw ClassicMiddleware) {
	m := mustGetMatcher(router, method)
	m.classicMiddlewares = append(m.classicMiddlewares, mw)
}

func MethodMiddleware[I any, O any](router *Router, method method, f TaskMiddlewareFn[O]) *TaskMiddleware[O] {
	m := mustGetMatcher(router, method)
	task := FnToTaskMiddleware(router, f)
	m.taskMiddlewares = append(m.taskMiddlewares, task)
	return task
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS
/////////////////////////////////////////////////////////////////////

var handlerTypes = struct {
	classic string
	task    string
}{
	classic: "classic",
	task:    "task",
}

type AnyRegisteredPattern interface {
	datafn.Any
	getHandlerType() string
	getClassicHandler() ClassicHandler
	getTaskHandler() tasks.AnyTask
	getClassicMiddlewares() []ClassicMiddleware
	getTaskMiddlewares() []tasks.AnyTask
}

type RegisteredPattern[I any, O any] struct {
	datafn.Any

	router  *Router
	method  string
	pattern string

	classicMiddlewares []ClassicMiddleware
	taskMiddlewares    []tasks.AnyTask

	handlerType    string
	classicHandler ClassicHandler
	taskHandler    tasks.AnyTask
}

func (rp *RegisteredPattern[I, O]) getHandlerType() string            { return rp.handlerType }
func (rp *RegisteredPattern[I, O]) getClassicHandler() ClassicHandler { return rp.classicHandler }
func (rp *RegisteredPattern[I, O]) getTaskHandler() tasks.AnyTask     { return rp.taskHandler }
func (rp *RegisteredPattern[I, O]) getClassicMiddlewares() []ClassicMiddleware {
	return rp.classicMiddlewares
}
func (rp *RegisteredPattern[I, O]) getTaskMiddlewares() []tasks.AnyTask { return rp.taskMiddlewares }

/////////////////////////////////////////////////////////////////////
/////// PATTERN HANDLERS
/////////////////////////////////////////////////////////////////////

func PatternClassic(
	router *Router, method method, pattern pattern, h ClassicHandler,
) {
	rp := &RegisteredPattern[any, any]{
		Any: datafn.ToPhantom[any, any](),

		router:  router,
		method:  method,
		pattern: pattern,

		handlerType:    handlerTypes.classic,
		classicHandler: h,
	}

	mustRegisterPattern(rp)
}

func PatternClassicFunc(
	router *Router, method method, pattern pattern, h ClassicHandlerFunc,
) {
	PatternClassic(router, method, pattern, h)
}

func Pattern[I any, O any](
	router *Router, method method, pattern pattern, f TaskHandlerFn[I, O],
) *RegisteredPattern[I, O] {
	task := FnToTaskHandler(router, f)

	rp := &RegisteredPattern[I, O]{
		Any: datafn.ToPhantom[I, O](),

		router:  router,
		method:  method,
		pattern: pattern,

		handlerType: handlerTypes.task,
		taskHandler: task,
	}

	mustRegisterPattern(rp)

	return rp
}

func PatternMiddlewareClassic[I any, O any](rp *RegisteredPattern[I, O], mw ClassicMiddleware) {
	rp.classicMiddlewares = append(rp.classicMiddlewares, mw)
}

func PatternMiddleware[PI any, PO any, MWO any](rp *RegisteredPattern[PI, PO], mw *TaskMiddleware[MWO]) None {
	rp.taskMiddlewares = append(rp.taskMiddlewares, mw)
	return None{}
}

package router

import (
	"encoding/json"
	"net/http"

	"github.com/sjc5/kit/pkg/datafn"
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/opt"
	"github.com/sjc5/kit/pkg/tasks"
)

// each method type gets its own matcher
// middleware can be applied (i) globally, (ii) per-method, or (iii) per-route

type RegisteredPattern struct {
	handlerType     string // "std" or "task"
	stdHandler      StdHandler
	taskHandler     tasks.Task
	stdMiddlewares  []StdMiddleware
	taskMiddlewares []tasks.Task
}

func (rp *RegisteredPattern) AddStdMiddleware(mw StdMiddleware) *RegisteredPattern {
	rp.stdMiddlewares = append(rp.stdMiddlewares, mw)
	return rp
}

func (rp *RegisteredPattern) AddTaskMiddleware(mw tasks.Task) *RegisteredPattern {
	rp.taskMiddlewares = append(rp.taskMiddlewares, mw)
	return rp
}

type decoratedMatcher struct {
	matcher            *matcher.Matcher
	stdMiddlewares     []StdMiddleware
	taskMiddlewares    []tasks.Task
	registeredPatterns map[Pattern]*RegisteredPattern
	typedCtxGetters    map[Pattern]typedCtxGetterWrapper
}

type Router struct {
	marshalInput       func(r *http.Request) any
	tasksRegistry      *tasks.Registry
	stdMiddlewares     []StdMiddleware
	taskMiddlewares    []tasks.Task
	methodToMatcherMap map[Method]*decoratedMatcher
	matcherOptions     *matcher.Options
	notFoundHandler    StdHandler
}

type Options struct {
	MarshalInput           func(r *http.Request) any
	TasksRegistry          *tasks.Registry
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.
}

func NewRouter(opts *Options) *Router {
	if opts.TasksRegistry == nil {
		panic("tasks registry is required")
	}

	matcherOptions := new(matcher.Options)

	if opts == nil {
		opts = new(Options)
	}
	matcherOptions.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	matcherOptions.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	return &Router{
		marshalInput:       opts.MarshalInput,
		tasksRegistry:      opts.TasksRegistry,
		methodToMatcherMap: make(map[Method]*decoratedMatcher),
		matcherOptions:     matcherOptions,
	}
}

func (router *Router) getMatcher(method string) (*decoratedMatcher, bool) {
	if _, ok := permittedHTTPMethods[method]; !ok {
		return nil, false
	}

	m, ok := router.methodToMatcherMap[method]
	if !ok {
		m = &decoratedMatcher{
			matcher:            matcher.New(router.matcherOptions),
			registeredPatterns: make(map[string]*RegisteredPattern),
			typedCtxGetters:    make(map[string]typedCtxGetterWrapper),
		}
		router.methodToMatcherMap[method] = m
	}

	return m, true
}

func (router *Router) MethodStd(method, pattern string, handler StdHandler) *RegisteredPattern {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.matcher.RegisterPattern(pattern)
	rp := &RegisteredPattern{handlerType: "std", stdHandler: handler}
	m.registeredPatterns[pattern] = rp

	return rp
}

func (router *Router) MethodStdFunc(method, pattern string, handlerFunc StdHandlerFunc) *RegisteredPattern {
	return router.MethodStd(method, pattern, handlerFunc)
}

type typedCtxGetterFunc[I any] func(r *http.Request, match *matcher.Match) *Ctx[I]

func (f typedCtxGetterFunc[I]) getTypedCtx(r *http.Request, match *matcher.Match) CtxMarker {
	return f(r, match)
}

type typedCtxGetterWrapper interface {
	getTypedCtx(r *http.Request, match *matcher.Match) CtxMarker
}

func Register[I any, O any](
	router *Router, method Method, pattern Pattern, handler datafn.Unwrapped[*Ctx[I], O],
) *RegisteredPattern {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.matcher.RegisterPattern(pattern)

	rp := &RegisteredPattern{
		handlerType: "task",
		taskHandler: tasks.New(
			router.tasksRegistry,
			func(c *tasks.CtxInput[*Ctx[I]]) (O, error) { return handler(c.Input) },
		),
	}

	m.typedCtxGetters[pattern] = typedCtxGetterFunc[I](func(r *http.Request, match *matcher.Match) *Ctx[I] {
		rCtx := new(Ctx[I])
		if len(match.Params) > 0 {
			rCtx.params = match.Params
		}
		if len(match.SplatValues) > 0 {
			rCtx.splatValues = match.SplatValues
		}
		rCtx.tasksCtx = router.tasksRegistry.NewCtxFromRequest(r)
		rCtx.input = router.marshalInput(rCtx.Request()).(I)
		return rCtx
	})

	m.registeredPatterns[pattern] = rp

	return rp
}

func (router *Router) MethodTask(method, pattern string, handler tasks.Task) *RegisteredPattern {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.matcher.RegisterPattern(pattern)
	rp := &RegisteredPattern{handlerType: "task", taskHandler: handler}
	m.registeredPatterns[pattern] = rp

	return rp
}

func (router *Router) AddGlobalStdMiddleware(mw StdMiddleware) *Router {
	router.stdMiddlewares = append(router.stdMiddlewares, mw)
	return router
}

func (router *Router) AddGlobalTaskMiddleware(mw tasks.Task) *Router {
	router.taskMiddlewares = append(router.taskMiddlewares, mw)
	return router
}

func (router *Router) AddStdMiddlewareToMethod(method string, mw StdMiddleware) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.stdMiddlewares = append(m.stdMiddlewares, mw)
	return router
}

func (router *Router) AddTaskMiddlewareToMethod(method string, mw tasks.Task) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.taskMiddlewares = append(m.taskMiddlewares, mw)
	return router
}

func (router *Router) AddStdMiddlewareToPattern(method, pattern string, mw StdMiddleware) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	rp, ok := m.registeredPatterns[pattern]
	if !ok {
		panic("pattern not registered")
	}

	rp.AddStdMiddleware(mw)
	return router
}

func (router *Router) AddTaskMiddlewareToPattern(method, pattern string, mw tasks.Task) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	rp, ok := m.registeredPatterns[pattern]
	if !ok {
		panic("pattern not registered")
	}

	rp.AddTaskMiddleware(mw)
	return router
}

func (router *Router) SetNotFoundHandler(handler StdHandler) *Router {
	router.notFoundHandler = handler
	return router
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m, ok := router.getMatcher(r.Method)
	if !ok {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bestMatch, ok := m.matcher.FindBestMatch(r.URL.Path)
	if !ok {
		if router.notFoundHandler != nil {
			router.notFoundHandler.ServeHTTP(w, r)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}

	bestMatchRegistered := m.registeredPatterns[bestMatch.Pattern()]
	typedCtxGetter := m.typedCtxGetters[bestMatch.Pattern()]

	ctx := r.Context()
	rCtx := typedCtxGetter.getTypedCtx(r, bestMatch)
	r = r.WithContext(ctx)

	if bestMatchRegistered.handlerType == "std" {
		handler := bestMatchRegistered.stdHandler
		handler = runAppropriateMiddlewares(router, rCtx, m, bestMatchRegistered, handler)
		handler.ServeHTTP(w, r)
	} else {
		handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results, ok := rCtx.TasksCtx().Run(&tasks.RunArg{
				Task:  bestMatchRegistered.taskHandler,
				Input: rCtx,
			})
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			data := bestMatchRegistered.taskHandler.GetTaskResult(results)
			if data == nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !data.OK() {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			json, err := json.Marshal(data.Data)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(json)
		})

		handler := http.Handler(handlerFunc)
		handler = runAppropriateMiddlewares(router, rCtx, m, bestMatchRegistered, handler)
		handler.ServeHTTP(w, r)
	}

}

func runAppropriateMiddlewares(
	router *Router,
	rCtx CtxMarker,
	methodMatcher *decoratedMatcher,
	bestMatch *RegisteredPattern,
	handler http.Handler,
) http.Handler {
	// Middlewares need to be chained backwards
	for i := len(bestMatch.stdMiddlewares) - 1; i >= 0; i-- { // pattern
		handler = bestMatch.stdMiddlewares[i](handler)
	}
	for i := len(methodMatcher.stdMiddlewares) - 1; i >= 0; i-- { // method
		handler = methodMatcher.stdMiddlewares[i](handler)
	}
	for i := len(router.stdMiddlewares) - 1; i >= 0; i-- { // global
		handler = router.stdMiddlewares[i](handler)
	}

	capacity := len(bestMatch.taskMiddlewares) + len(methodMatcher.taskMiddlewares) + len(router.taskMiddlewares)
	tasksToRun := make([]tasks.Task, 0, capacity)
	tasksToRun = append(tasksToRun, router.taskMiddlewares...)        // global
	tasksToRun = append(tasksToRun, methodMatcher.taskMiddlewares...) // method
	tasksToRun = append(tasksToRun, bestMatch.taskMiddlewares...)     // pattern

	runArgs := make([]*tasks.RunArg, 0, len(tasksToRun))
	for _, task := range tasksToRun {
		runArgs = append(runArgs, &tasks.RunArg{Task: task, Input: rCtx})
	}
	rCtx.TasksCtx().Run(runArgs...)

	return handler
}

type ctxKey string

const routerCtxKey ctxKey = "routerCtx"

type CtxMarker interface {
	getInput() any
	Params() Params
	SplatValues() []string
	TasksCtx() *tasks.Ctx
	Request() *http.Request
}

type Ctx[I any] struct {
	params      Params
	splatValues []string
	tasksCtx    *tasks.Ctx
	input       I
}

func (c *Ctx[I]) getInput() any          { return c.input }
func (c *Ctx[I]) Input() I               { return c.input }
func (c *Ctx[I]) Params() Params         { return c.params }
func (c *Ctx[I]) SplatValues() []string  { return c.splatValues }
func (c *Ctx[I]) TasksCtx() *tasks.Ctx   { return c.tasksCtx }
func (c *Ctx[I]) Request() *http.Request { return c.tasksCtx.Request() }

func getRouterCtx[I any](r *http.Request) *Ctx[I] {
	if ctx, ok := r.Context().Value(routerCtxKey).(*Ctx[I]); ok {
		return ctx
	}
	return nil
}

func GetParam[I any](r *http.Request, name string) string {
	return GetParams[I](r)[name]
}

func GetParams[I any](r *http.Request) Params {
	if routerCtx := getRouterCtx[I](r); routerCtx != nil {
		return routerCtx.params
	}
	return nil
}

func GetSplatValues[I any](r *http.Request) []string {
	if routerCtx := getRouterCtx[I](r); routerCtx != nil {
		return routerCtx.splatValues
	}
	return nil
}

func RunTask[I any, O any](c *Ctx[I], task tasks.TaskWithHelper[*Ctx[I], O]) (O, error) {
	return task.Run(c.TasksCtx(), c)
}

func NewTask[I any, O any](tasksRegistry *tasks.Registry, task func(*Ctx[I]) (O, error)) tasks.TaskWithHelper[*Ctx[I], O] {
	return tasks.New(tasksRegistry, func(c *tasks.CtxInput[*Ctx[I]]) (O, error) {
		return task(c.Input)
	})
}

package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/opt"
)

// each method type gets its own matcher
// middleware can be applied (i) globally, (ii) per-method, or (iii) per-route

type Handler = http.Handler
type HandlerFunc = http.HandlerFunc
type Middleware = func(Handler) Handler
type Params = matcher.Params

type RegisteredPattern struct {
	middlewares []Middleware
	handler     Handler
}

func (rp *RegisteredPattern) AddMiddleware(middleware Middleware) {
	rp.middlewares = append(rp.middlewares, middleware)
}

type decoratedMatcher struct {
	matcher            *matcher.Matcher
	middlewares        []Middleware
	registeredPatterns map[string]*RegisteredPattern
}

type methodToMatcherMap = map[string]*decoratedMatcher

var permittedHTTPMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodConnect: {},
	http.MethodOptions: {},
	http.MethodTrace:   {},
}

type Router struct {
	middlewares []Middleware
	methodToMatcherMap
	matcherOptions  *matcher.Options
	notFoundHandler Handler
}

type Options struct {
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.
}

func NewRouter(opts *Options) *Router {
	matcherOptions := new(matcher.Options)

	if opts == nil {
		opts = new(Options)
	}
	matcherOptions.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	matcherOptions.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	return &Router{
		methodToMatcherMap: make(methodToMatcherMap),
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
		}
		router.methodToMatcherMap[method] = m
	}

	return m, true
}

func (router *Router) Method(method, pattern string, handler Handler) *RegisteredPattern {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.matcher.RegisterPattern(pattern)
	rp := &RegisteredPattern{handler: handler}
	m.registeredPatterns[pattern] = rp

	return rp
}

func (router *Router) MethodFunc(method, pattern string, handlerFunc HandlerFunc) *RegisteredPattern {
	return router.Method(method, pattern, handlerFunc)
}

func (router *Router) AddGlobalMiddleware(middleware Middleware) *Router {
	router.middlewares = append(router.middlewares, middleware)
	return router
}

func (router *Router) AddMiddlewareToMethod(method string, middleware Middleware) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.middlewares = append(m.middlewares, middleware)
	return router
}

func (router *Router) AddMiddlewareToPattern(method, pattern string, middleware Middleware) *Router {
	m, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	m.registeredPatterns[pattern].middlewares = append(m.registeredPatterns[pattern].middlewares, middleware)
	return router
}

func (router *Router) SetNotFoundHandler(handler Handler) *Router {
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

	ctx := r.Context()
	var rCtx *routerCtx
	if len(bestMatch.Params) > 0 {
		rCtx = new(routerCtx)
		rCtx.params = bestMatch.Params
	}
	if len(bestMatch.SplatValues) > 0 {
		if rCtx == nil {
			rCtx = new(routerCtx)
		}
		rCtx.splatValues = bestMatch.SplatValues
	}
	r = r.WithContext(ctx)

	bestMatchRegistered := m.registeredPatterns[bestMatch.Pattern()]
	handler := bestMatchRegistered.handler

	// Middlewares need to be chained backwards
	for i := len(bestMatchRegistered.middlewares) - 1; i >= 0; i-- {
		handler = bestMatchRegistered.middlewares[i](handler)
	}
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}
	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}

	handler.ServeHTTP(w, r)
}

type ctxKey string

const routerCtxKey ctxKey = "routerCtx"

type routerCtx struct {
	params      Params
	splatValues []string
}

func getRouterCtx(r *http.Request) *routerCtx {
	if ctx, ok := r.Context().Value(routerCtxKey).(*routerCtx); ok {
		return ctx
	}
	return nil
}

func GetParam(r *http.Request, name string) string {
	return GetParams(r)[name]
}

func GetParams(r *http.Request) Params {
	if routerCtx := getRouterCtx(r); routerCtx != nil {
		return routerCtx.params
	}
	return nil
}

func GetSplatValues(r *http.Request) []string {
	if routerCtx := getRouterCtx(r); routerCtx != nil {
		return routerCtx.splatValues
	}
	return nil
}

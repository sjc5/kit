package router

import (
	"net/http"
)

// each method type gets its own matcher
// middleware can be applied (i) globally, (ii) per-method, or (iii) per-route

type Handler = http.Handler
type HandlerFunc = http.HandlerFunc
type Middleware = func(Handler) Handler

type methodToMatcherMap = map[string]*matcher

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
	matcherOptions  *MatcherOptions
	notFoundHandler Handler
}

type RouterOptions struct {
	// Optional. Defaults to '$'.
	DynamicParamPrefixRune rune

	// Optional. Defaults to '$'.
	SplatSegmentRune rune
}

func NewRouter(routerOptions *RouterOptions) *Router {
	matcherOptions := new(MatcherOptions)

	if routerOptions != nil {
		matcherOptions.DynamicParamPrefixRune = routerOptions.DynamicParamPrefixRune
		matcherOptions.SplatSegmentRune = routerOptions.SplatSegmentRune
	}

	return &Router{
		methodToMatcherMap: make(methodToMatcherMap),
		matcherOptions:     matcherOptions,
	}
}

func (router *Router) getMatcher(method string) (*matcher, bool) {
	if _, ok := permittedHTTPMethods[method]; !ok {
		return nil, false
	}

	matcher, ok := router.methodToMatcherMap[method]
	if !ok {
		matcher = NewMatcher(router.matcherOptions)
		router.methodToMatcherMap[method] = matcher
	}

	return matcher, true
}

func (router *Router) Handle(method, pattern string, handler Handler) *RegisteredPattern {
	matcher, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	return matcher.RegisterPattern(pattern).SetHandler(handler)
}

func (router *Router) HandleFunc(method, pattern string, handlerFunc HandlerFunc) *RegisteredPattern {
	return router.Handle(method, pattern, handlerFunc)
}

func (router *Router) AddGlobalMiddleware(middleware Middleware) *Router {
	router.middlewares = append(router.middlewares, middleware)
	return router
}

func (router *Router) AddMethodMiddleware(method string, middleware Middleware) *Router {
	matcher, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	matcher.AddMiddleware(middleware)
	return router
}

func (router *Router) AddMiddlewareToPattern(method, pattern string, middleware Middleware) *Router {
	matcher, ok := router.getMatcher(method)
	if !ok {
		panic("invalid HTTP method")
	}

	matcher.AddMiddlewareToPattern(pattern, middleware)
	return router
}

func (router *Router) SetNotFoundHandler(handler Handler) *Router {
	router.notFoundHandler = handler
	return router
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matcher, ok := router.getMatcher(r.Method)
	if !ok {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bestMatch, ok := matcher.FindBestMatch(r.URL.Path)
	if !ok {
		if router.notFoundHandler != nil {
			router.notFoundHandler.ServeHTTP(w, r)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}

	handler := bestMatch.handler

	// Middlewares need to be chained backwards
	for i := len(bestMatch.middlewares) - 1; i >= 0; i-- {
		handler = bestMatch.middlewares[i](handler)
	}
	for i := len(matcher.middlewares) - 1; i >= 0; i-- {
		handler = matcher.middlewares[i](handler)
	}
	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}

	handler.ServeHTTP(w, r)
}

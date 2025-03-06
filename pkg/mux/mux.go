package mux

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/contextutil"
	"github.com/sjc5/kit/pkg/genericsutil"
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/opt"
	"github.com/sjc5/kit/pkg/tasks"
)

// __TODO see if there are ways of consolidating signatures at all,
// or if task middlewares, standalone task handlers, and nested task
// handlers all really need inherently different types.

// __TODO add in response instructions concept, and make as consistent
// as possible across all types of tasks.

/////////////////////////////////////////////////////////////////////
/////// VARIOUS TYPES (CORE)
/////////////////////////////////////////////////////////////////////

type (
	HTTPMiddleware                = func(http.Handler) http.Handler
	TaskMiddlewareFunc[O any]     = genericsutil.IOFunc[*http.Request, O]
	TaskMiddleware[O any]         = tasks.RegisteredTask[*http.Request, O]
	TaskHandlerFunc[I any, O any] = genericsutil.IOFunc[*ReqData[I], O]
)

/////////////////////////////////////////////////////////////////////
/////// CORE ROUTER STRUCTURE
/////////////////////////////////////////////////////////////////////

type Router struct {
	_marshal_input         func(r *http.Request, iPtr any) error
	_tasks_registry        *tasks.Registry
	_http_mws              []HTTPMiddleware
	_task_mws              []tasks.AnyRegisteredTask
	_method_to_matcher_map map[string]*_Method_Matcher
	_matcher_opts          *matcher.Options
	_not_found_handler     http.Handler
}

type _Method_Matcher struct {
	_matcher          *matcher.Matcher
	_http_mws         []HTTPMiddleware
	_task_mws         []tasks.AnyRegisteredTask
	_routes           map[string]_Route_Marker
	_req_data_getters map[string]_Req_Data_Getter
}

/////////////////////////////////////////////////////////////////////
/////// NEW ROUTER
/////////////////////////////////////////////////////////////////////

type Options struct {
	MarshalInput           func(r *http.Request, inputPtr any) error
	TasksRegistry          *tasks.Registry
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.
}

func NewRouter(opts *Options) *Router {
	_matcher_opts := new(matcher.Options)

	if opts == nil {
		opts = new(Options)
	}

	_matcher_opts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	_matcher_opts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	return &Router{
		_marshal_input:         opts.MarshalInput,
		_tasks_registry:        opts.TasksRegistry,
		_method_to_matcher_map: make(map[string]*_Method_Matcher),
		_matcher_opts:          _matcher_opts,
	}
}

/////////////////////////////////////////////////////////////////////
/////// PUBLIC UTILITIES
/////////////////////////////////////////////////////////////////////

func TaskHandlerFromFunc[I any, O any](tasksRegistry *tasks.Registry, taskHandlerFunc TaskHandlerFunc[I, O]) *TaskHandler[I, O] {
	return tasks.Register(tasksRegistry, func(tasksCtx *tasks.Arg[*ReqData[I]]) (O, error) {
		return taskHandlerFunc(tasksCtx.Input)
	})
}

func TaskMiddlewareFromFunc[O any](tasksRegistry *tasks.Registry, taskMwFunc TaskMiddlewareFunc[O]) *TaskMiddleware[O] {
	return tasks.Register(tasksRegistry, func(tasksCtx *tasks.Arg[*http.Request]) (O, error) {
		return taskMwFunc(tasksCtx.Input)
	})
}

/////////////////////////////////////////////////////////////////////
/////// GLOBAL MIDDLEWARES
/////////////////////////////////////////////////////////////////////

func SetGlobalTaskMiddleware[O any](router *Router, taskMw *TaskMiddleware[O]) {
	router._task_mws = append(router._task_mws, taskMw)
}

func SetGlobalHTTPMiddleware(router *Router, httpMw HTTPMiddleware) {
	router._http_mws = append(router._http_mws, httpMw)
}

/////////////////////////////////////////////////////////////////////
/////// METHOD-LEVEL MIDDLEWARES
/////////////////////////////////////////////////////////////////////

func SetMethodLevelTaskMiddleware[I any, O any](
	router *Router, method string, taskMw TaskMiddleware[O],
) {
	_method_matcher := _must_get_matcher(router, method)
	_task := taskMw
	_method_matcher._task_mws = append(_method_matcher._task_mws, _task)
}

func SetMethodLevelHTTPMiddleware(router *Router, method string, httpMw HTTPMiddleware) {
	_method_matcher := _must_get_matcher(router, method)
	_method_matcher._http_mws = append(_method_matcher._http_mws, httpMw)
}

/////////////////////////////////////////////////////////////////////
/////// PATTERN-LEVEL MIDDLEWARE APPLIERS
/////////////////////////////////////////////////////////////////////

func SetRouteLevelTaskMiddleware[PI any, PO any, MWO any](route *Route[PI, PO], taskMw *TaskMiddleware[MWO]) {
	route._task_mws = append(route._task_mws, taskMw)
}

func SetPatternLevelHTTPMiddleware[I any, O any](route *Route[I, O], httpMw HTTPMiddleware) {
	route._http_mws = append(route._http_mws, httpMw)
}

/////////////////////////////////////////////////////////////////////
/////// NOT FOUND HANDLER
/////////////////////////////////////////////////////////////////////

func SetGlobalNotFoundHTTPHandler(router *Router, httpHandler http.Handler) {
	router._not_found_handler = httpHandler
}

/////////////////////////////////////////////////////////////////////
/////// HANDLER TYPES
/////////////////////////////////////////////////////////////////////

var _handler_types = struct {
	_http string
	_task string
}{
	_http: "http",
	_task: "task",
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (CORE)
/////////////////////////////////////////////////////////////////////

// Core registered pattern structure
type Route[I any, O any] struct {
	genericsutil.ZeroHelper[I, O]

	_router  *Router
	_method  string
	_pattern string

	_http_mws []HTTPMiddleware
	_task_mws []tasks.AnyRegisteredTask

	_handler_type string
	_http_handler http.Handler
	_task_handler tasks.AnyRegisteredTask
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (COBWEBS)
/////////////////////////////////////////////////////////////////////

// Interface to allow for type-agnostic handling of generic-typed routes.
type _Route_Marker interface {
	genericsutil.AnyZeroHelper
	_get_handler_type() string
	_get_http_handler() http.Handler
	_get_task_handler() tasks.AnyRegisteredTask
	_get_http_mws() []HTTPMiddleware
	_get_task_mws() []tasks.AnyRegisteredTask
}

// Implementing the routeMarker interface on the Route struct.
func (route *Route[I, O]) _get_handler_type() string                  { return route._handler_type }
func (route *Route[I, O]) _get_http_handler() http.Handler            { return route._http_handler }
func (route *Route[I, O]) _get_task_handler() tasks.AnyRegisteredTask { return route._task_handler }
func (route *Route[I, O]) _get_http_mws() []HTTPMiddleware {
	return route._http_mws
}
func (route *Route[I, O]) _get_task_mws() []tasks.AnyRegisteredTask { return route._task_mws }

/////////////////////////////////////////////////////////////////////
/////// CORE PATTERN REGISTRATION FUNCTIONS
/////////////////////////////////////////////////////////////////////

func RegisterTaskHandler[I any, O any](
	router *Router, method, pattern string, taskHandler *TaskHandler[I, O],
) *Route[I, O] {
	_route := _new_route_struct[I, O](router, method, pattern)
	_route._handler_type = _handler_types._task
	_route._task_handler = taskHandler
	_must_register_route(_route)
	return _route
}

func RegisterHandlerFunc(router *Router, method, pattern string, httpHandlerFunc http.HandlerFunc) *Route[any, any] {
	return RegisterHandler(router, method, pattern, httpHandlerFunc)
}

func RegisterHandler(router *Router, method, pattern string, httpHandler http.Handler) *Route[any, any] {
	_route := _new_route_struct[any, any](router, method, pattern)
	_route._handler_type = _handler_types._http
	_route._http_handler = httpHandler
	_must_register_route(_route)
	return _route
}

/////////////////////////////////////////////////////////////////////
/////// REQUEST DATA (COBWEBS)
/////////////////////////////////////////////////////////////////////

// Interface to allow for type-agnostic handling of generic-typed request data.
type _Req_Data_Marker interface {
	_get_input() any
	Params() Params
	SplatValues() []string
	TasksCtx() *tasks.TasksCtx
	Request() *http.Request
	_get_orig() any
}

// Implementing the reqDataMarker interface on the ReqData struct.
func (rd *ReqData[I]) _get_input() any           { return rd._input }
func (rd *ReqData[I]) Params() Params            { return rd._params }
func (rd *ReqData[I]) SplatValues() []string     { return rd._splat_vals }
func (rd *ReqData[I]) TasksCtx() *tasks.TasksCtx { return rd._tasks_ctx }
func (rd *ReqData[I]) Request() *http.Request    { return rd._tasks_ctx.Request() }
func (rd *ReqData[I]) _get_orig() any            { return rd }

type _Req_Data_Getter interface {
	_get_req_data(r *http.Request, match *matcher.Match) _Req_Data_Marker
}

type _Req_Data_Getter_Impl[I any] func(*http.Request, *matcher.Match) *ReqData[I]

func (f _Req_Data_Getter_Impl[I]) _get_req_data(r *http.Request, m *matcher.Match) _Req_Data_Marker {
	return f(r, m)
}

/////////////////////////////////////////////////////////////////////
/////// NATIVE CONTEXT
/////////////////////////////////////////////////////////////////////

// __TODO, not fully implemented yet, but the point of this is so that
// users can access the request data from http handlers. Not
// necessary for task handlers.

var _context_store = contextutil.NewStore[_Req_Data_Marker]("_req_data")

func GetReqData[I any](r *http.Request) *ReqData[I] {
	_req_data_marker := _context_store.GetValueFromContext(r.Context())
	return genericsutil.AssertOrZero[*ReqData[I]](_req_data_marker)
}

func GetParam[I any](r *http.Request, key string) string {
	return GetParams[I](r)[key]
}

func GetParams[I any](r *http.Request) Params {
	if _req_data := GetReqData[I](r); _req_data != nil {
		return _req_data._params
	}
	return nil
}

func GetSplatValues[I any](r *http.Request) []string {
	if _req_data := GetReqData[I](r); _req_data != nil {
		return _req_data._splat_vals
	}
	return nil
}

/////////////////////////////////////////////////////////////////////
/////// SERVE HTTP
/////////////////////////////////////////////////////////////////////

func (_router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_method_matcher, err := _get_matcher(_router, r.Method)
	if err != nil {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_match, ok := _method_matcher._matcher.FindBestMatch(r.URL.Path)
	if !ok {
		if _router._not_found_handler != nil {
			_router._not_found_handler.ServeHTTP(w, r)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}

	_orig_pattern := _match.OriginalPattern()
	_route := _method_matcher._routes[_orig_pattern]

	_req_data_getter, ok := _method_matcher._req_data_getters[_orig_pattern]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_req_data_marker := _req_data_getter._get_req_data(r, _match)

	r = _context_store.GetRequestWithContext(r, _req_data_marker)

	if _route._get_handler_type() == _handler_types._http {
		_handler := _route._get_http_handler()
		_handler = run_appropriate_mws(_router, _req_data_marker, _method_matcher, _route, _handler)
		_handler.ServeHTTP(w, r)
		return
	}

	_handler_func := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// __TODO need more flexible content types and http statuses

		_tasks_ctx := _req_data_marker.TasksCtx()

		_prepared_task := tasks.PrepAny(_tasks_ctx, _route._get_task_handler(), _req_data_marker._get_orig())
		if ok := _tasks_ctx.ParallelPreload(_prepared_task); !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_data, err := _prepared_task.GetAny()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_json, err := json.Marshal(_data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(_json)
	})

	_handler := http.Handler(_handler_func)
	_handler = run_appropriate_mws(_router, _req_data_marker, _method_matcher, _route, _handler)
	_handler.ServeHTTP(w, r)
}

func run_appropriate_mws(
	_router *Router,
	_req_data_marker _Req_Data_Marker,
	_method_matcher *_Method_Matcher,
	_route_marker _Route_Marker,
	_handler http.Handler,
) http.Handler {

	/////// HTTP MIDDLEWARES

	// Middlewares need to be chained backwards
	_http_mws := _route_marker._get_http_mws()
	for i := len(_http_mws) - 1; i >= 0; i-- { // pattern
		_handler = _http_mws[i](_handler)
	}
	for i := len(_method_matcher._http_mws) - 1; i >= 0; i-- { // method
		_handler = _method_matcher._http_mws[i](_handler)
	}
	for i := len(_router._http_mws) - 1; i >= 0; i-- { // global
		_handler = _router._http_mws[i](_handler)
	}

	/////// TASK MIDDLEWARES

	_task_mws := _route_marker._get_task_mws()
	_cap := len(_task_mws) + len(_method_matcher._task_mws) + len(_router._task_mws)
	_tasks_to_run := make([]tasks.AnyRegisteredTask, 0, _cap)
	_tasks_to_run = append(_tasks_to_run, _router._task_mws...)         // global
	_tasks_to_run = append(_tasks_to_run, _method_matcher._task_mws...) // method
	_tasks_to_run = append(_tasks_to_run, _task_mws...)                 // pattern

	_tasks_ctx := _req_data_marker.TasksCtx()
	_tasks_with_input := make([]tasks.AnyPreparedTask, 0, len(_tasks_to_run))
	for _, task := range _tasks_to_run {
		_tasks_with_input = append(_tasks_with_input, tasks.PrepAny(_tasks_ctx, task, _req_data_marker._get_orig()))
	}
	_tasks_ctx.ParallelPreload(_tasks_with_input...)

	return _handler
}

/////////////////////////////////////////////////////////////////////
/////// INTERNAL HELPERS
/////////////////////////////////////////////////////////////////////

func _new_route_struct[I any, O any](_router *Router, _method, _pattern string) *Route[I, O] {
	return &Route[I, O]{_router: _router, _method: _method, _pattern: _pattern}
}

func _must_register_route[I any, O any](_route *Route[I, O]) {
	_method_matcher := _must_get_matcher(_route._router, _route._method)
	_method_matcher._matcher.RegisterPattern(_route._pattern)
	_method_matcher._routes[_route._pattern] = _route
	_method_matcher._req_data_getters[_route._pattern] = _to_req_data_getter_impl(_route)
}

func _req_data_starter[I any](_match *matcher.Match, _tasks_registry *tasks.Registry, r *http.Request) *ReqData[I] {
	_req_data := new(ReqData[I])
	if len(_match.Params) > 0 {
		_req_data._params = _match.Params
	}
	if len(_match.SplatValues) > 0 {
		_req_data._splat_vals = _match.SplatValues
	}
	_req_data._tasks_ctx = _tasks_registry.NewCtxFromRequest(r)
	return _req_data
}

func _to_req_data_getter_impl[I any, O any](_route *Route[I, O]) _Req_Data_Getter_Impl[I] {
	return _Req_Data_Getter_Impl[I](
		func(r *http.Request, _match *matcher.Match) *ReqData[I] {
			_req_data := _req_data_starter[I](_match, _route._router._tasks_registry, r)
			_input_ptr := _route.IPtr()
			if _route._router._marshal_input != nil {
				if err := _route._router._marshal_input(_req_data.Request(), _input_ptr); err != nil {
					// __TODO do something here
					fmt.Println("validation err", err)
				}
			}
			_req_data._input = *(_input_ptr.(*I))
			return _req_data
		},
	)
}

func _must_get_matcher(_router *Router, _method string) *_Method_Matcher {
	_method_matcher, err := _get_matcher(_router, _method)
	if err != nil {
		panic(err)
	}
	return _method_matcher
}

func _get_matcher(_router *Router, _method string) (*_Method_Matcher, error) {
	if _, ok := _permitted_http_methods[_method]; !ok {
		return nil, errors.New("unknown method")
	}
	_method_matcher, ok := _router._method_to_matcher_map[_method]
	if !ok {
		_method_matcher = &_Method_Matcher{
			_matcher:          matcher.New(_router._matcher_opts),
			_routes:           make(map[string]_Route_Marker),
			_req_data_getters: make(map[string]_Req_Data_Getter),
		}
		_router._method_to_matcher_map[_method] = _method_matcher
	}
	return _method_matcher, nil
}

var _permitted_http_methods = map[string]struct{}{
	http.MethodGet: {}, http.MethodHead: {}, // query methods
	http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {}, http.MethodDelete: {}, // mutation methods
	http.MethodConnect: {}, http.MethodOptions: {}, http.MethodTrace: {}, // other methods
}

package mux

import (
	"net/http"

	"github.com/sjc5/kit/pkg/genericsutil"
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/opt"
	"github.com/sjc5/kit/pkg/tasks"
)

/////////////////////////////////////////////////////////////////////
/////// CORE ROUTER STRUCTURE
/////////////////////////////////////////////////////////////////////

// Always a GET / no input parsing / all tasks
type NestedRouter struct {
	_tasks_registry *tasks.Registry
	_matcher        *matcher.Matcher
	_routes         map[string]AnyNestedRoute
}

func (nr *NestedRouter) AllRoutes() map[string]AnyNestedRoute {
	return nr._routes
}

func (nr *NestedRouter) TasksRegistry() *tasks.Registry {
	return nr._tasks_registry
}

/////////////////////////////////////////////////////////////////////
/////// NEW ROUTER
/////////////////////////////////////////////////////////////////////

type NestedOptions struct {
	TasksRegistry          *tasks.Registry
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.

	// Optional. Defaults to empty string (trailing slash in your patterns).
	// You can set it to something like "_index" to make it explicit.
	ExplicitIndexSegment string
}

func NewNestedRouter(opts *NestedOptions) *NestedRouter {
	_matcher_opts := new(matcher.Options)

	if opts == nil {
		opts = new(NestedOptions)
	}

	if opts.TasksRegistry == nil {
		panic("tasks registry is required for nested router")
	}

	_matcher_opts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	_matcher_opts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')
	_matcher_opts.ExplicitIndexSegment = opt.Resolve(opts, opts.ExplicitIndexSegment, "")

	return &NestedRouter{
		_tasks_registry: opts.TasksRegistry,
		_matcher:        matcher.New(_matcher_opts),
		_routes:         make(map[string]AnyNestedRoute),
	}
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (CORE)
/////////////////////////////////////////////////////////////////////

type NestedRoute[O any] struct {
	genericsutil.ZeroHelper[None, O]

	_router  *NestedRouter
	_pattern string

	_task_handler tasks.AnyRegisteredTask
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (COBWEBS)
/////////////////////////////////////////////////////////////////////

type AnyNestedRoute interface {
	genericsutil.AnyZeroHelper
	_get_task_handler() tasks.AnyRegisteredTask
	Pattern() string
}

func (route *NestedRoute[O]) _get_task_handler() tasks.AnyRegisteredTask { return route._task_handler }
func (route *NestedRoute[O]) Pattern() string                            { return route._pattern }

/////////////////////////////////////////////////////////////////////
/////// CORE PATTERN REGISTRATION FUNCTIONS
/////////////////////////////////////////////////////////////////////

func RegisterNestedTaskHandler[O any](
	router *NestedRouter, pattern string, taskHandler *TaskHandler[None, O],
) {
	_route := _new_nested_route_struct[O](router, pattern)
	_route._task_handler = taskHandler
	_must_register_nested_route(_route)
}

func RegisterNestedPatternWithoutHandler(router *NestedRouter, pattern string) {
	_route := _new_nested_route_struct[None](router, pattern)
	_must_register_nested_route(_route)
}

/////////////////////////////////////////////////////////////////////
/////// REQUEST DATA (CORE)
/////////////////////////////////////////////////////////////////////

type NestedReqData = ReqData[None]

/////////////////////////////////////////////////////////////////////
/////// RUN NESTED TASKS
/////////////////////////////////////////////////////////////////////

type NestedTasksResult struct {
	_pattern string
	_data    any
	_err     error
}

func (ntr *NestedTasksResult) Pattern() string { return ntr._pattern }
func (ntr *NestedTasksResult) OK() bool        { return ntr._err == nil }
func (ntr *NestedTasksResult) Data() any       { return ntr._data }
func (ntr *NestedTasksResult) Err() error      { return ntr._err }

type NestedTasksResults struct {
	Params      Params
	SplatValues []string
	Map         map[string]*NestedTasksResult
	Slice       []*NestedTasksResult
}

// Second return value (bool) indicates matches found, not success of tasks run
func RunNestedTasks(nestedRouter *NestedRouter, tasksCtx *tasks.TasksCtx, r *http.Request) (*NestedTasksResults, bool) {
	_matches, ok := nestedRouter._matcher.FindNestedMatches(r.URL.Path)
	if !ok {
		return nil, false
	}

	_last_match := _matches[len(_matches)-1]

	_results := new(NestedTasksResults)
	_results.Params = _last_match.Params
	_results.SplatValues = _last_match.SplatValues

	// Initialize result containers up front
	_results.Map = make(map[string]*NestedTasksResult, len(_matches))
	_results.Slice = make([]*NestedTasksResult, len(_matches))

	// First, identify which matches have tasks that need to be run
	_tasks_with_input := make([]tasks.AnyPreparedTask, 0)
	_task_indices := make(map[int]int) // Maps match index to task index

	for i, _match := range _matches {
		_nested_route_marker, routeExists := nestedRouter._routes[_match.OriginalPattern()]

		// Create result object regardless of whether a task exists
		_res := &NestedTasksResult{_pattern: _match.OriginalPattern()}
		_results.Map[_match.OriginalPattern()] = _res
		_results.Slice[i] = _res

		// Skip task preparation if route doesn't exist or has no task handler
		if !routeExists {
			continue
		}

		_task := _nested_route_marker._get_task_handler()
		if _task == nil {
			// User can register patterns without a handler so that they still match
			continue
		}

		_rd := &ReqData[None]{
			_params:     _last_match.Params,
			_splat_vals: _last_match.SplatValues,
			_tasks_ctx:  tasksCtx,
			_input:      None{},
		}

		_tasks_with_input = append(_tasks_with_input, tasks.PrepAny(tasksCtx, _task, _rd))
		_task_indices[i] = len(_tasks_with_input) - 1 // Store the mapping between match index and task index
	}

	// Only run parallelPreload if we have tasks to run
	if len(_tasks_with_input) > 0 {
		tasksCtx.ParallelPreload(_tasks_with_input...)
	}

	// Process task results for matches that had tasks
	for matchIdx, taskIdx := range _task_indices {
		_res := _results.Slice[matchIdx]
		_data, err := _tasks_with_input[taskIdx].GetAny()
		_res._data = _data
		_res._err = err
	}

	return _results, true
}

/////////////////////////////////////////////////////////////////////
/////// INTERNAL HELPERS
/////////////////////////////////////////////////////////////////////

func _new_nested_route_struct[O any](_router *NestedRouter, _pattern string) *NestedRoute[O] {
	return &NestedRoute[O]{_router: _router, _pattern: _pattern}
}

func _must_register_nested_route[O any](_route *NestedRoute[O]) {
	_matcher := _route._router._matcher
	_matcher.RegisterPattern(_route._pattern)
	_route._router._routes[_route._pattern] = _route
}

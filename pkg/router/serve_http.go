package router

import (
	"encoding/json"
	"net/http"

	"github.com/sjc5/kit/pkg/tasks"
)

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	methodMatcher, err := getMatcher(router, r.Method)
	if err != nil {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	match, ok := methodMatcher.matcher.FindBestMatch(r.URL.Path)
	if !ok {
		if router.notFoundHandler != nil {
			router.notFoundHandler.ServeHTTP(w, r)
			return
		} else {
			http.NotFound(w, r)
			return
		}
	}

	rp := methodMatcher.registeredPatterns[match.Pattern()]

	routerCtx := methodMatcher.ctxGetters[match.Pattern()].getCtx(r, match)
	r = addRouterCtxToNativeContext(r, routerCtx)

	if rp.getHandlerType() == handlerTypes.classic {
		handler := rp.getClassicHandler()
		handler = runAppropriateMiddlewares(router, routerCtx, methodMatcher, rp, handler)
		handler.ServeHTTP(w, r)
		return
	}

	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// __TODO need more flexible content types and http statuses

		tasksCtx := routerCtx.TasksCtx()

		preparedTask := tasks.PrepAny(tasksCtx, rp.getTaskHandler(), routerCtx)
		if ok := tasksCtx.ParallelPreload(preparedTask); !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data, err := preparedTask.GetAny()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		json, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	})

	handler := http.Handler(handlerFunc)
	handler = runAppropriateMiddlewares(router, routerCtx, methodMatcher, rp, handler)
	handler.ServeHTTP(w, r)
}

func runAppropriateMiddlewares(
	router *Router,
	routerCtx CtxMarker,
	methodMatcher *decoratedMatcher,
	rp AnyRegisteredPattern,
	handler http.Handler,
) http.Handler {

	/////// CLASSIC MIDDLEWARES

	// Middlewares need to be chained backwards
	classicMiddlewares := rp.getClassicMiddlewares()
	for i := len(classicMiddlewares) - 1; i >= 0; i-- { // pattern
		handler = classicMiddlewares[i](handler)
	}
	for i := len(methodMatcher.classicMiddlewares) - 1; i >= 0; i-- { // method
		handler = methodMatcher.classicMiddlewares[i](handler)
	}
	for i := len(router.classicMiddlewares) - 1; i >= 0; i-- { // global
		handler = router.classicMiddlewares[i](handler)
	}

	/////// TASK MIDDLEWARES

	taskMiddlewares := rp.getTaskMiddlewares()
	capacity := len(taskMiddlewares) + len(methodMatcher.taskMiddlewares) + len(router.taskMiddlewares)
	tasksToRun := make([]tasks.AnyTask, 0, capacity)
	tasksToRun = append(tasksToRun, router.taskMiddlewares...)        // global
	tasksToRun = append(tasksToRun, methodMatcher.taskMiddlewares...) // method
	tasksToRun = append(tasksToRun, taskMiddlewares...)               // pattern

	tasksCtx := routerCtx.TasksCtx()
	tasksWithInput := make([]tasks.AnyTaskWithInput, 0, len(tasksToRun))
	for _, task := range tasksToRun {
		tasksWithInput = append(tasksWithInput, tasks.PrepAny(tasksCtx, task, routerCtx))
	}
	tasksCtx.ParallelPreload(tasksWithInput...)

	return handler
}

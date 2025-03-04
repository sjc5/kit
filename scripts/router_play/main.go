package main

import (
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/router"
	"github.com/sjc5/kit/pkg/tasks"
	"github.com/sjc5/kit/pkg/validate"
)

var tasksRegistry = tasks.NewRegistry()

var r = router.NewRouter(&router.Options{
	TasksRegistry: tasksRegistry,
	MarshalInput:  validate.New().URLSearchParamsInto,
})

func main() {
	server := &http.Server{Addr: ":9090", Handler: r}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

type None struct{}

func Get[I any, O any](p string, f router.TaskHandlerFn[I, O]) *router.Route[I, O] {
	return router.RegisterTaskHandler(r, "GET", p, f)
}
func Post[I any, O any](p string, f router.TaskHandlerFn[I, O]) *router.Route[I, O] {
	return router.RegisterTaskHandler(r, "POST", p, f)
}

/////////////////////////////////////////////////////////////////////////////////////////////

var AuthTask = router.TaskMiddlewareFromFn(r, func(_ *http.Request) (string, error) {
	fmt.Println("running auth ...")
	return "auth-token-43892", nil
})

var _ = router.SetGlobalTaskMiddleware(r, AuthTask)

var sallyPattern = Get("/sally", func(rc *router.ReqData[string]) (string, error) {
	fmt.Println("running sally ...", rc)
	someInput := rc.Input()
	fmt.Println("running sally 2 ...", someInput)
	return "sally", nil
})

type Test struct {
	Input string `json:"input"`
}

var catchAllRoute = Get("/*", func(rc *router.ReqData[Test]) (map[string]string, error) {
	input := rc.Input()
	tc := rc.TasksCtx()
	token, _ := AuthTask.Prep(tc, rc.Request()).Get()
	fmt.Println("Auth token from catch route:", token)

	fmt.Println("running hello ...", rc.SplatValues(), sallyPattern.Phantom)
	return map[string]string{
		"hello": "world",
		"foo":   input.Input,
	}, nil
})

var _ = router.SetRouteLevelTaskMiddleware(catchAllRoute, AuthTask)

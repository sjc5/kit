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

/////////////////////////////////////////////////////////////////////////////////////////////

var AuthTask = router.FnToTaskMiddleware(r, func(_ *http.Request) (string, error) {
	fmt.Println("running auth ...")
	return "auth-token-43892", nil
})

var _ = router.GlobalMiddleware(r, AuthTask)

var sallyPattern = router.Pattern(r, "GET", "/sally", func(rc *router.RouterCtx[string]) (string, error) {
	fmt.Println("running sally ...", rc)
	someInput := rc.Input()
	fmt.Println("running sally 2 ...", someInput)
	return "sally", nil
})

type Test struct {
	Input string `json:"input"`
}

var catchAllRoute = router.Pattern(r, "GET", "/*", func(rc *router.RouterCtx[Test]) (map[string]string, error) {
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

var _ = router.PatternMiddleware(catchAllRoute, AuthTask)

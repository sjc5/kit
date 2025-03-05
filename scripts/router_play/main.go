package main

import (
	"fmt"
	"io"
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
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	fmt.Println("Server running on port 9090")

	// hit a certain path on the running server

	resp, err := http.Get("http://localhost:9090/")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response status:", resp.Status)

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Response body:", string(bodyText))
}

type None struct{}

func Get[I any, O any](p string, f router.TaskHandlerFn[I, O]) *router.Route[I, O] {
	return router.RegisterTaskHandler(r, "GET", p, f)
}
func Post[I any, O any](p string, f router.TaskHandlerFn[I, O]) *router.Route[I, O] {
	return router.RegisterTaskHandler(r, "POST", p, f)
}

/////////////////////////////////////////////////////////////////////////////////////////////

type Test struct {
	Input string `json:"input"`
}

// var AuthTask = router.TaskMiddlewareFromFn(r, func(_ *http.Request) (string, error) {
// 	fmt.Println("running auth ...")
// 	return "auth-token-43892", nil
// })

// var _ = router.SetGlobalTaskMiddleware(r, AuthTask)

var _ = Get("", func(rd *router.ReqData[Test]) (string, error) {
	fmt.Println("running empty str ...", rd.Request().URL.Path)
	return "empty str", nil
})

// var _ = Get("/", func(rd *router.ReqData[Test]) (string, error) {
// 	fmt.Println("running slash ...", rd.Request().URL.Path)
// 	return "slash", nil
// })

// var sallyPattern = Get("/sally", func(rd *router.ReqData[string]) (string, error) {
// 	fmt.Println("running sally ...", rd)
// 	someInput := rd.Input()
// 	fmt.Println("running sally 2 ...", someInput)
// 	return "sally", nil
// })

// var catchAllRoute = Get("/*", func(rd *router.ReqData[Test]) (map[string]string, error) {
// 	input := rd.Input()
// 	tc := rd.TasksCtx()
// 	token, _ := AuthTask.Prep(tc, rd.Request()).Get()
// 	fmt.Println("Auth token from catch route:", token)

// 	fmt.Println("running hello ...", rd.SplatValues(), sallyPattern.Phantom)
// 	return map[string]string{
// 		"hello": "world",
// 		"foo":   input.Input,
// 	}, nil
// })

// var _ = router.SetRouteLevelTaskMiddleware(catchAllRoute, AuthTask)

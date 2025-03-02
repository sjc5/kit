package main

import (
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/datafn"
	"github.com/sjc5/kit/pkg/router"
	"github.com/sjc5/kit/pkg/tasks"
)

var tasksRegistry = tasks.NewRegistry()

var r = router.NewRouter(&router.Options{
	TasksRegistry: tasksRegistry,
	MarshalInput: func(r *http.Request) any {
		return r.URL.Query().Get("input")
	},
})

func Register[I any, O any](m, p string, h datafn.Unwrapped[*router.Ctx[I], O]) *router.RegisteredPattern {
	return router.Register(r, m, p, h)
}

func Task[I any, O any](task func(*router.Ctx[I]) (O, error)) tasks.Task[*router.Ctx[I], O] {
	return router.NewTask(tasksRegistry, task)
}

func main() {
	server := &http.Server{Addr: ":9090", Handler: r}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

type None struct{}

/////////////////////////////////////////////////////////////////////////////////////////////

var AuthMW = Task(func(c *router.Ctx[None]) (string, error) {
	fmt.Println("running auth ...")
	return "auth-token-43892", nil
})

var Sally = Register("GET", "/sally", func(c *router.Ctx[string]) (string, error) {
	fmt.Println("running sally ...", c)
	someInput := c.Input()
	fmt.Println("running sally 2 ...", someInput)
	return "sally", nil
})

var CatchRoute = Register("GET", "/*", func(c *router.Ctx[None]) (map[string]string, error) {

	token, _ := router.RunTask(c, AuthMW)

	fmt.Println("Auth token from catch route:", token)

	fmt.Println("running hello ...", c.SplatValues())
	return map[string]string{
		"hello": "world",
		"foo":   "bar",
	}, nil
})

var _ = CatchRoute.AddTaskMiddleware(AuthMW).AddTaskMiddleware(AuthMW).AddTaskMiddleware(AuthMW)

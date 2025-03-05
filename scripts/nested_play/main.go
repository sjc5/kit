package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/router"
	"github.com/sjc5/kit/pkg/tasks"
)

var tasksRegistry = tasks.NewRegistry()

var r = router.NewNestedRouter(&router.NestedOptions{
	TasksRegistry:        tasksRegistry,
	ExplicitIndexSegment: "_index",
})

var _ = router.RegisterNestedTaskHandler(r, "/auth", func(rd *router.NestedReqData) (int, error) {
	fmt.Println("running auth   ...", rd.Request().URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth   ...", rd.Request().URL, time.Now().UnixMilli())
	return 123, nil
})

var _ = router.RegisterNestedTaskHandler(r, "/auth/larry", func(rd *router.NestedReqData) (int, error) {
	fmt.Println("running auth larry ...", rd.Request().URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth larry ...", rd.Request().URL, time.Now().UnixMilli())
	return 24892498, nil
	return 0, errors.New("auth larry error")
})

var _ = router.RegisterNestedTaskHandler(r, "/auth/larry/:id", func(rd *router.NestedReqData) (string, error) {
	fmt.Println("running auth larry :id ...", rd.Request().URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth larry :id ...", rd.Params()["id"], time.Now().UnixMilli())
	return "*** Larry has an ID of " + rd.Params()["id"], nil
})

func main() {
	req, _ := http.NewRequest("GET", "/auth/larry/12879", nil)

	tasksCtx := tasksRegistry.NewCtxFromRequest(req)

	results, _ := router.RunNestedTasks(r, tasksCtx, req)

	fmt.Println()

	fmt.Println("results.Params", results.Params)
	fmt.Println("results.SplatValues", results.SplatValues)

	for k, v := range results.PatternMap {
		fmt.Println()

		fmt.Println("result: ", k)

		if v.OK() {
			fmt.Println("Data: ", v.Data())
		} else {
			fmt.Println("Err : ", v.Err())
		}
	}

	fmt.Println()
}

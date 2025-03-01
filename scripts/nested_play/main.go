package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/nestedrouter"
)

var router = nestedrouter.New(nil)

func SetLoader[O any](pattern string, loader nestedrouter.Loader[O]) *nestedrouter.Router {
	return nestedrouter.RegisterPatternWithLoader(router, pattern, loader)
}

var _ = SetLoader("/auth", func(c *nestedrouter.Ctx) (int, error) {
	fmt.Println("running auth   ...", c.Req.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth   ...", c.Req.URL, time.Now().UnixMilli())
	return 123, nil
})

var _ = SetLoader("/auth/larry", func(c *nestedrouter.Ctx) (int, error) {
	fmt.Println("running auth larry ...", c.Req.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth larry ...", c.Req.URL, time.Now().UnixMilli())
	return 0, errors.New("auth larry error")
})

var _ = SetLoader("/auth/larry/:id", func(c *nestedrouter.Ctx) (string, error) {
	fmt.Println("running auth larry :id ...", c.Req.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Second)
	fmt.Println("finishing auth larry :id ...", c.Params["id"], time.Now().UnixMilli())
	return "*** Larry has an ID of " + c.Params["id"], nil
})

func main() {
	req, _ := http.NewRequest("GET", "/auth/larry/12879", nil)
	c := router.NewCtx(req)

	results, _ := c.GetLoaderResultsMap()
	for k, v := range results {
		fmt.Print("result: ", k)
		if v.Err != nil {
			fmt.Print(" | error: ", v.Err)
		} else {
			fmt.Print(" | data: ", v.Data)
		}
		fmt.Println()
	}
}

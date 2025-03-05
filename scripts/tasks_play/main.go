package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/tasks"
)

var registry = tasks.NewRegistry()

var Auth = tasks.New(registry, func(c *tasks.TasksCtxWithInput[any]) (int, error) {
	fmt.Println("running auth   ...", c.Request().URL, time.Now().UnixMilli())

	// return 0, errors.New("auth error")

	time.Sleep(1 * time.Second)
	fmt.Println("auth done", time.Now().UnixMilli())

	return 123, nil
})

var User = tasks.New(registry, func(c *tasks.TasksCtxWithInput[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running user   ...", user_id, time.Now().UnixMilli())

	// time.Sleep(500 * time.Millisecond)
	// c.Cancel()

	auth := Auth.Prep(c.TasksCtx, nil)

	if ok := c.ParallelPreload(auth); !ok {
		return "", errors.New("auth error")
	}

	token, _ := auth.Get()

	time.Sleep(1 * time.Second)
	fmt.Println("user done", time.Now().UnixMilli())

	return fmt.Sprintf("user-%d", token), nil
})

var User2 = tasks.New(registry, func(c *tasks.TasksCtxWithInput[string]) (string, error) {
	fmt.Println("running user2  ...", c.Request().URL, time.Now().UnixMilli())

	auth := Auth.Prep(c.TasksCtx, nil)

	if ok := c.ParallelPreload(auth); !ok {
		return "", errors.New("auth error")
	}

	token, _ := auth.Get()

	time.Sleep(1 * time.Second)
	fmt.Println("user2 done", time.Now().UnixMilli())

	return fmt.Sprintf("user2-%d", token), nil
})

var Profile = tasks.New(registry, func(c *tasks.TasksCtxWithInput[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running profile...", c.Request().URL, time.Now().UnixMilli())

	user := User.Prep(c.TasksCtx, user_id)
	user2 := User2.Prep(c.TasksCtx, user_id)

	if ok := c.ParallelPreload(user, user2); !ok {
		return "", errors.New("user error")
	}

	userData, _ := user.Get()
	user2Data, _ := user2.Get()

	time.Sleep(1 * time.Second)
	fmt.Println("profile done", time.Now().UnixMilli(), userData, user2Data)

	return "profile", nil
})

func main() {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	c := registry.NewCtxFromRequest(req)

	data, err := Profile.Prep(c, "32isdoghj").Get()

	fmt.Println("from main -- profile data:", data)
	fmt.Println("from main -- profile err:", err)
}

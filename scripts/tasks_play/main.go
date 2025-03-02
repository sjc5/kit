package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/tasks"
)

type CtxInput[I any] = tasks.CtxInput[I]

var registry = tasks.NewRegistry()

var Auth = tasks.New(registry, func(c *CtxInput[any]) (int, error) {
	fmt.Println("running auth   ...", c.Request().URL, time.Now().UnixMilli())

	// return 0, errors.New("auth error")

	time.Sleep(1 * time.Second)
	fmt.Println("auth done", time.Now().UnixMilli())

	return 123, nil
})

var User = tasks.New(registry, func(c *CtxInput[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running user   ...", user_id, time.Now().UnixMilli())

	// time.Sleep(500 * time.Millisecond)
	// c.Cancel()

	results, ok := c.Run(Auth.Input(nil))
	if !ok {
		return "", errors.New("auth error")
	}

	token := Auth.From(results)

	time.Sleep(1 * time.Second)
	fmt.Println("user done", time.Now().UnixMilli())

	return fmt.Sprintf("user-%d", token), nil
})

var User2 = tasks.New(registry, func(c *CtxInput[string]) (string, error) {
	fmt.Println("running user2  ...", c.Request().URL, time.Now().UnixMilli())

	results, ok := c.Run(Auth.Input(nil))
	if !ok {
		return "", errors.New("auth error")
	}

	token := Auth.From(results)

	time.Sleep(1 * time.Second)
	fmt.Println("user2 done", time.Now().UnixMilli())

	return fmt.Sprintf("user2-%d", token), nil
})

var Profile = tasks.New(registry, func(c *CtxInput[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running profile...", c.Request().URL, time.Now().UnixMilli())

	results, ok := c.Run(
		User.Input(user_id),
		User2.Input(user_id),
	)
	if !ok {
		return "", errors.New("user error")
	}

	user := User.From(results)
	user2 := User2.From(results)

	time.Sleep(1 * time.Second)
	fmt.Println("profile done", time.Now().UnixMilli(), user, user2)

	return "profile", nil
})

func main() {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	c := registry.NewCtxFromRequest(req)

	data, err := Profile.Run(c, "32isdoghj")

	fmt.Println("from main -- profile data:", data)
	fmt.Println("from main -- profile err:", err)
}

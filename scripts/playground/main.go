package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/tasks"
)

type Input = *http.Request
type Ctx = tasks.Ctx[Input]
type PreReqs = tasks.PreReqs

var graph = tasks.NewGraph(func(input Input) context.Context {
	return input.Context()
})

var Auth = tasks.New(graph, PreReqs{}, func(c *Ctx) (int, error) {
	fmt.Println("running auth   ...", c.Input.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Millisecond)
	// return 0, errors.New("auth error")
	return 123, nil
})

var User = tasks.New(graph, PreReqs{Auth}, func(c *Ctx) (string, error) {
	fmt.Println("running user   ...", c.Input.URL, time.Now().UnixMilli())
	time.Sleep(500 * time.Microsecond)
	c.Cancel()
	token, _ := Auth.GetOutput(c)
	return fmt.Sprintf("user-%d", token), nil
})

var User2 = tasks.New(graph, PreReqs{Auth}, func(c *Ctx) (string, error) {
	fmt.Println("running user2  ...", c.Input.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Millisecond)
	token, _ := Auth.GetOutput(c)
	return fmt.Sprintf("user2-%d", token), nil
})

var Profile = tasks.New(graph, PreReqs{Auth, User, User2}, func(c *Ctx) (string, error) {
	fmt.Println("running profile...", c.Input.URL, time.Now().UnixMilli())
	time.Sleep(1 * time.Millisecond)
	auth := Auth.GetPreReqOutput(c)
	user := User.GetPreReqOutput(c)
	user2 := User2.GetPreReqOutput(c)
	fmt.Println(auth, user, user2)
	return "profile", nil
})

func main() {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	c := graph.NewCtx(req)
	c.Run(Profile, Auth, User, User2)

	data, err := Profile.GetOutput(c)
	fmt.Println("data:", data)
	fmt.Println("err:", err)
}

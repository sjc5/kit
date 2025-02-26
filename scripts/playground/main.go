package main

import (
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/dep"
)

type Ctx = dep.Ctx[*http.Request]
type PreReqs = dep.PreReqs

var t = dep.NewTree[Ctx]()

var Auth = dep.New(t, PreReqs{}, func(c *Ctx) (string, error) {
	fmt.Println("running auth...", c.Input.URL)
	return "0x123", nil
})

var User = dep.New(t, PreReqs{User2}, func(c *Ctx) (string, error) {
	fmt.Println("running user...")
	token, _ := Auth.Get(c)
	return fmt.Sprintf("user-%s", token), nil
})

var User2 = dep.New(t, PreReqs{Auth}, func(c *Ctx) (int, error) {
	fmt.Println("running user2...")
	return 1, nil
})

var Profile = dep.New(t, PreReqs{User, User2}, func(c *Ctx) (string, error) {
	fmt.Println("running profile...")
	return "profile", nil
})

func main() {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	c := t.Ctx(req)
	c.LoadInParallel(Profile, Auth, User, User2)
	fmt.Println(Profile.Get(c))
	fmt.Println(User.Get(c))
	fmt.Println(User2.Get(c))
	fmt.Println(Auth.Get(c))
}

package nestedrouter

import (
	"net/http"
	"sync"

	"github.com/sjc5/kit/pkg/tasks"
)

type Ctx struct {
	mu *sync.Mutex

	Req         *http.Request
	Params      Params
	SplatValues []string

	router   *Router
	tasksCtx *tasks.Ctx
	matches  Matches
}

func newCtx(router *Router, r *http.Request) *Ctx {
	return &Ctx{
		mu:       &sync.Mutex{},
		Req:      r,
		router:   router,
		tasksCtx: router.tasksRegistry.NewCtxFromRequest(r),
	}
}

func (router *Router) NewCtx(r *http.Request) *Ctx {
	return newCtx(router, r)
}

func (c *Ctx) TasksCtx() *tasks.Ctx {
	return c.tasksCtx
}

func (c *Ctx) FindMatches() (Matches, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.matches != nil {
		return c.matches, true
	}

	var ok bool

	c.matches, ok = c.router.matcher.FindNestedMatches(c.Req.URL.Path)
	if !ok {
		return nil, false
	}

	return c.matches, true
}

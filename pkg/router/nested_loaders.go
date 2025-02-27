package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

type TaskCtxInput = *NestedRequestCtx

type NestedRequestCtx struct {
	Req         *http.Request
	Params      Params
	SplatValues []string

	nr      *NestedRouter
	taskCtx *tasks.Ctx[TaskCtxInput]
}

type NestedRouter struct {
	matcher    *matcher.Matcher
	tasksGraph *tasks.Graph[TaskCtxInput]
	loaders    map[string]tasks.Task
}

// __TODO take in opts
func NewNestedRouter() *NestedRouter {
	return &NestedRouter{
		matcher: matcher.New(nil),
		tasksGraph: tasks.NewGraph(func(ctx TaskCtxInput) context.Context {
			return ctx.Req.Context()
		}),
		loaders: make(map[string]tasks.Task),
	}
}

func newNestedRequestCtx(nr *NestedRouter, r *http.Request) *NestedRequestCtx {
	nrc := &NestedRequestCtx{Req: r, nr: nr}
	taskCtx := nr.tasksGraph.NewCtx(nrc)
	nrc.taskCtx = taskCtx
	return nrc
}

func AddLoader[O any](nr *NestedRouter, pattern string, loader tasks.Task) {
	nr.matcher.RegisterPattern(pattern)
	nr.loaders[pattern] = loader
}

func (nr *NestedRouter) RunParallelLoaders(r *http.Request) (*NestedRequestCtx, error) {
	c := newNestedRequestCtx(nr, r)

	matches, ok := nr.matcher.FindNestedMatches(r.URL.Path)
	if !ok {
		return nil, fmt.Errorf("no matches found")
	}

	lastMatch := matches[len(matches)-1]
	c.Params = lastMatch.Params
	c.SplatValues = lastMatch.SplatValues

	// Collect loaders and unique dependencies
	loaders := make([]tasks.Task, 0)
	for _, match := range matches {
		if loader, ok := nr.loaders[match.Pattern()]; ok {
			loaders = append(loaders, loader)
		}
	}

	c.taskCtx.Run(loaders...)

	return c, nil
}

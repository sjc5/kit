package nestedrouter

import (
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

type Params = matcher.Params
type Options = matcher.Options
type Match = matcher.Match
type Matches = []*Match

type Router struct {
	matcher       *matcher.Matcher
	tasksRegistry *tasks.Registry
	loaders       map[string]tasks.Task
	queries       map[string]tasks.Task
	mutations     map[string]tasks.Task
}

func New(opts *Options) *Router {
	return &Router{
		matcher:       matcher.New(opts),
		tasksRegistry: tasks.NewRegistry(),
		loaders:       make(map[string]tasks.Task),
		queries:       make(map[string]tasks.Task),
		mutations:     make(map[string]tasks.Task),
	}
}

func (rp *RegisteredPattern) Pattern() string {
	return rp.matcherRP.Pattern()
}

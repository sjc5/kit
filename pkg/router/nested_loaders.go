package router

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/sjc5/kit/pkg/parallel"
)

type NestedRequestCtx struct {
	Req          *http.Request
	Params       Params
	SplatValues  []string
	parallelExec *parallel.Executor

	mu            sync.Mutex
	nr            *NestedRouter
	depResults    map[string]any
	loadOnce      map[string]*sync.Once
	loadersData   map[string]any
	loadersErrors map[string]error
}

// Goals: maximize parallelism, minimize wasted work
// Achieved through dependency tracking, goroutines, and context cancellation

type depRunner func(ctx *NestedRequestCtx) (any, error)

type dep struct {
	parentDeps []string
	runner     depRunner
}

type NestedRouter struct {
	matcher *Matcher
	deps    map[string]*dep
	loaders map[string]*loaderWrapper
}

func NewNestedRouter(matcher *Matcher) *NestedRouter {
	return &NestedRouter{
		matcher: matcher,
		deps:    make(map[string]*dep),
		loaders: make(map[string]*loaderWrapper),
	}
}

func (nr *NestedRouter) AddDependency(key string, runner depRunner, parentDeps ...string) {
	nr.deps[key] = &dep{parentDeps: parentDeps, runner: runner}
}

func newNestedRequestCtx(nr *NestedRouter, r *http.Request) *NestedRequestCtx {
	return &NestedRequestCtx{
		Req:           r,
		parallelExec:  parallel.New(r.Context()),
		nr:            nr,
		depResults:    make(map[string]any),
		loadOnce:      make(map[string]*sync.Once),
		loadersData:   make(map[string]any),
		loadersErrors: make(map[string]error),
	}
}

func (c *NestedRequestCtx) setDepResult(key string, result any) {
	c.mu.Lock()
	c.depResults[key] = result
	c.mu.Unlock()
}

func (c *NestedRequestCtx) getLoadOnce(key string) *sync.Once {
	c.mu.Lock()
	defer c.mu.Unlock()
	once, exists := c.loadOnce[key]
	if !exists {
		once = &sync.Once{}
		c.loadOnce[key] = once
	}
	return once
}

func (c *NestedRequestCtx) depKeysToTasks(keys []string) []parallel.Task {
	dedupeMap := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		dedupeMap[key] = struct{}{}
	}

	tasks := make([]parallel.Task, 0, len(dedupeMap))
	for k := range dedupeMap {
		tasks = append(tasks, func(ctx context.Context) error {
			_, err := c.getDependencyData(k)
			return err
		})
	}

	return tasks
}

func (c *NestedRequestCtx) warmParentDependenciesInParallel(dep *dep) {
	c.parallelExec.RunCancelOnErr(c.depKeysToTasks(dep.parentDeps)...)
}

func (c *NestedRequestCtx) getDependencyData(key string) (any, error) {
	var err error
	c.getLoadOnce(key).
		Do(func() {
			var dep *dep
			var exists bool
			if dep, exists = c.nr.deps[key]; !exists {
				err = fmt.Errorf("dependency not found: %s", key)
				return
			}
			c.warmParentDependenciesInParallel(dep)
			err = c.parallelExec.RunCancelOnErr(func(ctx context.Context) error {
				var res any
				var err error
				if res, err = dep.runner(c); err != nil {
					return err
				}
				c.setDepResult(key, res)
				return nil

			})
		})
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	result := c.depResults[key]
	c.mu.Unlock()
	return result, nil
}

type loaderWrapper struct {
	deps   []string
	loader loader
}

type Loader[O any] func(ctx *NestedRequestCtx) (*O, error)

func (f Loader[O]) getInputInstance() any  { return nil }
func (f Loader[O]) getOutputInstance() any { return new(O) }

func (f Loader[O]) execute(c *NestedRequestCtx, dest *any) error {
	return c.parallelExec.RunNoCancelOnErr(func(ctx context.Context) error {
		result, err := f(c)
		if err != nil {
			return err
		}
		*dest = *result
		return nil
	})
}

func AddLoader[O any](nr *NestedRouter, pattern string, loader Loader[O], deps ...string) {
	nr.matcher.RegisterPattern(pattern)
	nr.loaders[pattern] = &loaderWrapper{deps: deps, loader: loader}
}

type loader interface {
	getInputInstance() any
	getOutputInstance() any
	execute(ctx *NestedRequestCtx, dest *any) error
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
	loaders := make(map[string]*loaderWrapper, len(matches))
	depKeys := []string{}
	for _, match := range matches {
		if loader, exists := nr.loaders[match.Pattern()]; exists {
			loaders[match.Pattern()] = loader
			depKeys = append(depKeys, loader.deps...)
		}
	}

	// Run dependencies in parallel, respecting order via getDependencyData
	if err := c.parallelExec.RunNoCancelOnErr(c.depKeysToTasks(depKeys)...); err != nil {
		return nil, err
	}

	// Run loaders in parallel
	loaderTasks := make([]parallel.Task, 0, len(loaders))
	for p, l := range loaders {
		loaderTasks = append(loaderTasks, func(ctx context.Context) error {
			data := l.loader.getOutputInstance()
			err := l.loader.execute(c, &data)
			c.mu.Lock()
			c.loadersData[p] = data
			c.loadersErrors[p] = err
			c.mu.Unlock()
			return err
		})
	}

	if err := c.parallelExec.RunNoCancelOnErr(loaderTasks...); err != nil {
		return nil, err
	}

	return c, nil
}

func GetData[T any](ctx *NestedRequestCtx, key string) (T, error) {
	data, err := ctx.getDependencyData(key)
	if err != nil {
		var t T
		return t, err
	}
	return data.(T), nil
}

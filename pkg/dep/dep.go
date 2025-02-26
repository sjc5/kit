package dep

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

type Runner[I any, O any] func(*Ctx[I]) (O, error)

func (r Runner[I, O]) run(c any) (any, error) { // implements runnerWrapper
	return r(c.(*Ctx[I]))
}

type runnerWrapper interface {
	run(any) (any, error)
}

type dep[I any] struct {
	parentDeps map[int]PreReq
	runner     runnerWrapper
}

func (d dep[I]) hasParents() bool {
	return len(d.parentDeps) > 0
}

type Tree[I any] struct {
	count int
	deps  map[int]*dep[I]
}

func (t *Tree[I]) Ctx(input I) *Ctx[I] {
	return newCtx(t, input)
}

func NewTree[C Ctx[I], I any]() *Tree[I] {
	return &Tree[I]{deps: make(map[int]*dep[I])}
}

type PreReq interface {
	getKey() int
}

type PreReqs = []PreReq

type TypedCtx interface {
	isAnyCtx()
}

type Dep[O any] interface {
	Get(TypedCtx) (O, error)
	getKey() int
}

// implements Dep and PreReq
type getterWrapper[I any, O any] struct {
	key int
	f   func(*Ctx[I]) (O, error)
}

func (g getterWrapper[I, O]) Get(c TypedCtx) (O, error) {
	return g.f(c.(*Ctx[I]))
}

func (g getterWrapper[I, O]) getKey() int {
	return g.key
}

func New[I any, O any](t *Tree[I], preReqs PreReqs, runner Runner[I, O]) Dep[O] {
	key := t.count
	t.count++

	parentDepsSet := make(map[int]PreReq, len(preReqs))
	for _, preReq := range preReqs {
		parentDepsSet[preReq.getKey()] = preReq
	}
	t.deps[key] = &dep[I]{parentDeps: parentDepsSet, runner: runner}

	return getterWrapper[I, O]{
		key: key,
		f: func(c *Ctx[I]) (O, error) {
			return get[I, O](c, key)
		},
	}
}

type Result struct {
	Data any
	Err  error

	once *sync.Once
}

func newResult() *Result {
	return &Result{once: &sync.Once{}}
}

func (r *Result) OK() bool {
	return r.Err == nil
}

type Results struct {
	results map[int]*Result
}

func newResults() *Results {
	return &Results{results: make(map[int]*Result)}
}

func (r Results) OK() bool {
	for _, result := range r.results {
		if !result.OK() {
			return false
		}
	}
	return true
}

type Ctx[I any] struct {
	Input   I
	mu      sync.Mutex
	tree    *Tree[I]
	results *Results
}

func (c *Ctx[I]) isAnyCtx() {}

func newCtx[I any](tree *Tree[I], input I) *Ctx[I] {
	return &Ctx[I]{
		Input:   input,
		tree:    tree,
		results: newResults(),
	}
}

func (c *Ctx[I]) LoadInParallel(keys ...PreReq) {
	var wg sync.WaitGroup
	for _, key := range keys {
		wg.Add(1)
		go func(key PreReq) {
			c.Load(key)
			wg.Done()
		}(key)
	}
	wg.Wait()
}

func (c *Ctx[I]) getLoadOnce(key int) *sync.Once {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results.results[key]
	if !ok {
		result = newResult()
		c.results.results[key] = result
	}
	return result.once
}

func (c *Ctx[I]) load(key int) {
	c.getLoadOnce(key).Do(func() {
		var dep *dep[I]
		var exists bool

		c.mu.Lock()
		if dep, exists = c.tree.deps[key]; !exists {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		if dep.hasParents() {
			c.LoadInParallel(slices.Collect(maps.Values(dep.parentDeps))...)
		}

		data, err := dep.runner.run(c)
		c.mu.Lock()
		c.results.results[key].Data = data
		c.results.results[key].Err = err
		c.mu.Unlock()
	})
}

func (c *Ctx[I]) Load(key PreReq) {
	c.load(key.getKey())
}

func get[I any, O any](c *Ctx[I], key int) (O, error) {
	c.load(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results.results[key]
	if !ok {
		var o O
		return o, fmt.Errorf("dependency not found: %d", key)
	}
	return result.Data.(O), result.Err
}

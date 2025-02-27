package tasks

// The awesome thing about this package is that it's automatically protected
// from circular deps by Go's 'compile-time "initialization cycle" errors.
// It uses an async DAG to run tasks with maximum concurrency, with a shared
// cancellable context.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
)

// __TODO I think we need to be checking for cancel in more places

type Task interface {
	getID() int
}

type resultHelper[O any] interface {
	GetOutput(ctx) (O, error)
	GetPreReqOutput(ctx) O
}

type TaskWithHelper[O any] interface {
	Task
	resultHelper[O]
}

type taskImpl[I any, O any] struct { // implements Task interface
	id int
	f  func(*Ctx[I]) (O, error)
}

func (g taskImpl[I, O]) getID() int { // implements task interface
	return g.id
}
func (g taskImpl[I, O]) GetOutput(c ctx) (O, error) { // implements resultHelper interface
	return g.f(c.(*Ctx[I]))
}
func (g taskImpl[I, O]) GetPreReqOutput(c ctx) O { // implements resultHelper interface
	o, _ := g.GetOutput(c)
	return o
}

// second argument to tasks.New
type PreReqs = []Task

// adds a new task to the graph
func New[I any, O any](t *Graph[I], preReqs PreReqs, f TaskFunc[I, O]) TaskWithHelper[O] {
	key := t.count
	t.count++

	dedupedPreReqs := make(map[int]Task, len(preReqs))
	for _, preReq := range preReqs {
		dedupedPreReqs[preReq.getID()] = preReq
	}
	t.graph[key] = &node[I]{preReqs: dedupedPreReqs, taskHelper: f}

	return taskImpl[I, O]{
		id: key,
		f: func(c *Ctx[I]) (O, error) {
			return get[I, O](c, key)
		},
	}
}

type TaskFunc[I any, O any] func(*Ctx[I]) (O, error) // implements taskHelper

func (r TaskFunc[I, O]) run(c any) (any, error) {
	return r(c.(*Ctx[I]))
}
func (r TaskFunc[I, O]) getInputZeroValue() any {
	var input I
	return input
}
func (r TaskFunc[I, O]) getOutputZeroValue() any {
	var o O
	return o
}

type taskHelper interface {
	run(any) (any, error)
	getInputZeroValue() any
	getOutputZeroValue() any
}

func get[I any, O any](c *Ctx[I], key int) (O, error) {
	c.run(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results.results[key]
	if !ok {
		var o O
		return o, fmt.Errorf("task result not found: %d", key)
	}
	return result.Data.(O), result.Err
}

/////////////////////////////////////////////////////////////////////
/////// GRAPH
/////////////////////////////////////////////////////////////////////

type node[I any] struct {
	preReqs    map[int]Task
	taskHelper taskHelper
}

func (d node[I]) hasPreReqs() bool {
	return len(d.preReqs) > 0
}

type Graph[I any] struct {
	count      int
	graph      map[int]*node[I]
	getContext func(I) context.Context
}

func (t *Graph[I]) NewCtx(input I) *Ctx[I] {
	return newCtx(t, input)
}

func NewGraph[C Ctx[I], I any](getContext func(I) context.Context) *Graph[I] {
	return &Graph[I]{
		graph:      make(map[int]*node[I]),
		getContext: getContext,
	}
}

/////////////////////////////////////////////////////////////////////
/////// CTX
/////////////////////////////////////////////////////////////////////

type ctx interface {
	isCtx()
}

type Ctx[I any] struct {
	Input I

	mu      *sync.Mutex
	graph   *Graph[I]
	results *TaskResultRegistry

	context context.Context
	cancel  context.CancelFunc
}

func (c *Ctx[I]) isCtx() {} // implements ctx interface

func newCtx[I any](graph *Graph[I], input I) *Ctx[I] {
	contextWithCancel, cancel := context.WithCancel(graph.getContext(input))
	return &Ctx[I]{
		Input:   input,
		mu:      &sync.Mutex{},
		graph:   graph,
		results: newResults(),
		context: contextWithCancel,
		cancel:  cancel,
	}
}

func (c *Ctx[I]) Context() context.Context {
	return c.context
}

func (c *Ctx[I]) Cancel() {
	c.cancel()
}

func (c *Ctx[I]) Run(tasks ...Task) {
	if len(tasks) == 0 {
		return
	}

	if len(tasks) == 1 {
		c.run(tasks[0].getID())
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, t := range tasks {
		go func() {
			c.Run(t)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (c *Ctx[I]) run(taskID int) {
	var node *node[I]
	var exists bool

	c.mu.Lock()
	if _, ok := c.results.results[taskID]; !ok {
		c.results.results[taskID] = &TaskResult{once: &sync.Once{}}
	}
	if node, exists = c.graph.graph[taskID]; !exists {
		panic("node not found")
	}
	c.mu.Unlock()

	if c.context.Err() != nil {
		c.mu.Lock()
		c.results.results[taskID].Data = node.taskHelper.getOutputZeroValue()
		c.results.results[taskID].Err = errors.New("parent context canceled before execution")
		c.mu.Unlock()
		return
	}

	c.getSyncOnce(taskID).Do(func() {
		if node.hasPreReqs() {
			c.Run(slices.Collect(maps.Values(node.preReqs))...)
		}

		c.mu.Lock()
		// check if context is canceled
		if c.context.Err() != nil {
			c.results.results[taskID].Data = node.taskHelper.getOutputZeroValue()
			c.results.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
			return
		}

		// check that all parents are OK
		for _, p := range node.preReqs {
			if !c.results.results[p.getID()].OK() {
				c.results.results[taskID].Data = node.taskHelper.getOutputZeroValue()
				c.results.results[taskID].Err = errors.New("parent task failed")
				c.mu.Unlock()
				return
			}
		}
		c.mu.Unlock()

		resultChan := make(chan *TaskResult, 1)
		go func() {
			data, err := node.taskHelper.run(c)
			resultChan <- &TaskResult{Data: data, Err: err}
		}()

		select {
		case <-c.context.Done():
			c.mu.Lock()
			c.results.results[taskID].Data = node.taskHelper.getOutputZeroValue()
			c.results.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
		case result := <-resultChan:
			c.mu.Lock()
			c.results.results[taskID].Data = result.Data
			c.results.results[taskID].Err = result.Err
			c.mu.Unlock()
		}
	})
}

func (c *Ctx[I]) getSyncOnce(taskID int) *sync.Once {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results.results[taskID]
	if !ok {
		result = newTaskResult()
		c.results.results[taskID] = result
	}
	return result.once
}

/////////////////////////////////////////////////////////////////////
/////// RESULTS
/////////////////////////////////////////////////////////////////////

type TaskResult struct {
	Data any
	Err  error
	once *sync.Once
}

func newTaskResult() *TaskResult {
	return &TaskResult{once: &sync.Once{}}
}

func (r *TaskResult) OK() bool {
	return r.Err == nil
}

type TaskResultRegistry struct {
	results map[int]*TaskResult
}

func newResults() *TaskResultRegistry {
	return &TaskResultRegistry{results: make(map[int]*TaskResult)}
}

func (r TaskResultRegistry) AllOK() bool {
	for _, result := range r.results {
		if !result.OK() {
			return false
		}
	}
	return true
}

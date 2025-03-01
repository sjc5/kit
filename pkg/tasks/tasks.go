package tasks

// One awesome thing about this package is that, due to how it is structured,
// it is automatically protected from circular deps by Go's 'compile-time
// "initialization cycle" errors.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type Task interface {
	getID() int
	GetTaskResult(*PreLoadResult) *TaskResult
	GetOutputZeroValue() any
}

type resultHelper[I any, O any] interface {
	Run(ctx, I) (O, error)
}

type TaskWithHelper[I any, O any] interface {
	Task
	resultHelper[I, O]
	Input(I) *RunArg
	From(*PreLoadResult) O
}

type taskImpl[I any, O any] struct { // implements TaskWithHelper interface
	id int
	f  func(*Ctx, I) (O, error)
}

func (g taskImpl[I, O]) getID() int { // implements Task interface
	return g.id
}
func (g taskImpl[I, O]) Run(c ctx, input I) (O, error) { // implements resultHelper interface
	return g.f(c.(*Ctx), input)
}
func (g taskImpl[I, O]) From(r *PreLoadResult) O {
	var zero I
	o, _ := g.f(r.ctx, zero)
	return o
}
func (g taskImpl[I, O]) GetTaskResult(r *PreLoadResult) *TaskResult {
	var zero I
	o, err := g.f(r.ctx, zero)
	return &TaskResult{Data: o, Err: err}
}
func (g taskImpl[I, O]) GetOutputZeroValue() any {
	var o O
	return o
}

// adds a new task to the registry
func New[I any, O any](tr *Registry, f TaskFunc[I, O]) TaskWithHelper[I, O] {
	id := tr.count
	tr.count++

	tr.registry[id] = f

	return taskImpl[I, O]{
		id: id,
		f: func(c *Ctx, input I) (O, error) {
			c.run(id, input)
			c.mu.Lock()
			defer c.mu.Unlock()
			result, ok := c.results[id]
			if !ok {
				var o O
				return o, fmt.Errorf("task result not found for task with id: %d", id)
			}
			return result.Data.(O), result.Err
		},
	}
}

type TaskFunc[I any, O any] func(*Ctx, I) (O, error) // implements taskHelper

func (r TaskFunc[I, O]) run(c any, input any) (any, error) {
	var zero I
	_, ok := input.(I)
	if !ok {
		return r(c.(*Ctx), zero)
	}
	return r(c.(*Ctx), input.(I))
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
	run(any, any) (any, error)
	getInputZeroValue() any
	getOutputZeroValue() any
}

/////////////////////////////////////////////////////////////////////
/////// TASKS REGISTRY
/////////////////////////////////////////////////////////////////////

type Registry struct {
	count    int
	registry map[int]taskHelper
}

func (tr *Registry) NewCtxFromNativeContext(parentContext context.Context) *Ctx {
	return newCtx(tr, parentContext, nil)
}

func (tr *Registry) NewCtxFromRequest(r *http.Request) *Ctx {
	return newCtx(tr, r.Context(), r)
}

func NewRegistry() *Registry {
	return &Registry{registry: make(map[int]taskHelper)}
}

/////////////////////////////////////////////////////////////////////
/////// CTX
/////////////////////////////////////////////////////////////////////

type ctx interface {
	isCtx()
}

type Ctx struct {
	mu       *sync.Mutex
	request  *http.Request
	registry *Registry
	results  TaskResults

	context context.Context
	cancel  context.CancelFunc
}

func (c *Ctx) isCtx() {} // implements ctx interface

func newCtx(registry *Registry, parentContext context.Context, r *http.Request) *Ctx {
	contextWithCancel, cancel := context.WithCancel(parentContext)
	return &Ctx{
		mu:       &sync.Mutex{},
		request:  r,
		registry: registry,
		results:  newResults(),
		context:  contextWithCancel,
		cancel:   cancel,
	}
}

func (c *Ctx) Request() *http.Request {
	return c.request
}

func (c *Ctx) Context() context.Context {
	return c.context
}

func (c *Ctx) Cancel() {
	c.cancel()
}

func (task taskImpl[I, O]) Input(input I) *RunArg {
	return &RunArg{Task: task, Input: input}
}

type RunArg struct {
	Task  Task
	Input any
}

type PreLoadResult struct {
	ctx *Ctx
}

func (r *PreLoadResult) OK() bool {
	return r.ctx.results.AllOK()
}

func (c *Ctx) Run(tasks ...*RunArg) (*PreLoadResult, bool) {
	if len(tasks) == 0 {
		return &PreLoadResult{ctx: c}, true
	}

	if len(tasks) == 1 {
		t := tasks[0]
		c.run(t.Task.getID(), t.Input)
		return &PreLoadResult{ctx: c}, c.results.AllOK()
	}

	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, t := range tasks {
		go func() {
			c.run(t.Task.getID(), t.Input)
			wg.Done()
		}()
	}
	wg.Wait()

	return &PreLoadResult{ctx: c}, c.results.AllOK()
}

func (c *Ctx) run(taskID int, input any) {
	taskHelper := c.registry.registry[taskID]

	c.mu.Lock()
	if _, ok := c.results[taskID]; !ok {
		c.results[taskID] = &TaskResult{once: &sync.Once{}}
	}
	c.mu.Unlock()

	if c.context.Err() != nil {
		c.mu.Lock()
		c.results[taskID].Data = taskHelper.getOutputZeroValue()
		c.results[taskID].Err = errors.New("parent context canceled before execution")
		c.mu.Unlock()
		return
	}

	c.getSyncOnce(taskID).Do(func() {
		// check if context is canceled
		if c.context.Err() != nil {
			c.mu.Lock()
			c.results[taskID].Data = taskHelper.getOutputZeroValue()
			c.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
			return
		}

		resultChan := make(chan *TaskResult, 1)
		go func() {
			data, err := taskHelper.run(c, input)
			resultChan <- &TaskResult{Data: data, Err: err}
		}()

		select {
		case <-c.context.Done():
			c.mu.Lock()
			c.results[taskID].Data = taskHelper.getOutputZeroValue()
			c.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
		case result := <-resultChan:
			c.mu.Lock()
			c.results[taskID].Data = result.Data
			c.results[taskID].Err = result.Err
			c.mu.Unlock()
		}
	})
}

func (c *Ctx) getSyncOnce(taskID int) *sync.Once {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results[taskID]
	if !ok {
		result = newTaskResult()
		c.results[taskID] = result
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

type TaskResults map[int]*TaskResult

func newResults() TaskResults {
	return TaskResults(make(map[int]*TaskResult))
}

func (results TaskResults) AllOK() bool {
	for _, result := range results {
		if !result.OK() {
			return false
		}
	}
	return true
}

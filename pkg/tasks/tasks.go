package tasks

// A "Task", as used in this package, is simply a function that takes in input,
// returns data (or an error), and runs a maximum of one time per exection
// context, even if invoked repeatedly.
//
// One cool thing about this package is that, due to how it is structured, it
// is automatically protected from circular deps by Go's 'compile-time
// "initialization cycle" errors.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/sjc5/kit/pkg/datafn"
)

// second argument to tasks.New(registry, taskFn)
type Fn[I any, O any] = datafn.Fn[*TasksCtxWithInput[I], O]

// sole argument to TaskFn
type TasksCtxWithInput[I any] struct {
	Input I
	*TasksCtx
}

type AnyTask interface {
	datafn.Any

	getID() int
}

// returned from tasks.New(registry, taskFn)
type Task[I any, O any] struct {
	Fn[I, O] // implements AnyTask

	id int
}

// implements AnyTask
func (g Task[I, O]) getID() int { return g.id }

// adds a new task to the registry
func New[I any, O any](tr *Registry, f Fn[I, O]) *Task[I, O] {
	id := tr.count
	tr.count++

	tr.registry[id] = datafn.CtxFnToWrapped(func(ctx *TasksCtx, input I) (O, error) {
		return f(&TasksCtxWithInput[I]{TasksCtx: ctx, Input: input})
	})

	return &Task[I, O]{
		id: id,
		Fn: func(c *TasksCtxWithInput[I]) (O, error) {
			c.TasksCtx.doOnce(id, c.TasksCtx, c.Input)
			c.TasksCtx.mu.Lock()
			defer c.TasksCtx.mu.Unlock()
			result, ok := c.TasksCtx.results.results[id]
			if !ok {
				var o O
				return o, fmt.Errorf("task result not found for task with id: %d", id)
			}
			return result.Data.(O), result.Err
		},
	}
}

/////////////////////////////////////////////////////////////////////
/////// TASKS REGISTRY
/////////////////////////////////////////////////////////////////////

type Registry struct {
	count    int
	registry map[int]datafn.AnyCtxFnWrapped
}

func (tr *Registry) NewCtxFromNativeContext(parentContext context.Context) *TasksCtx {
	return newTasksCtx(tr, parentContext, nil)
}

func (tr *Registry) NewCtxFromRequest(r *http.Request) *TasksCtx {
	return newTasksCtx(tr, r.Context(), r)
}

func NewRegistry() *Registry {
	return &Registry{registry: make(map[int]datafn.AnyCtxFnWrapped)}
}

/////////////////////////////////////////////////////////////////////
/////// CTX
/////////////////////////////////////////////////////////////////////

type TasksCtx struct {
	mu       *sync.Mutex
	request  *http.Request
	registry *Registry
	results  *TaskResults

	context context.Context
	cancel  context.CancelFunc
}

func newTasksCtx(registry *Registry, parentContext context.Context, r *http.Request) *TasksCtx {
	contextWithCancel, cancel := context.WithCancel(parentContext)

	c := &TasksCtx{
		mu:       &sync.Mutex{},
		request:  r,
		registry: registry,
		context:  contextWithCancel,
		cancel:   cancel,
	}

	c.results = newResults(c)

	return c
}

func (c *TasksCtx) Request() *http.Request {
	return c.request
}

func (c *TasksCtx) NativeContext() context.Context {
	return c.context
}

func (c *TasksCtx) CancelNativeContext() {
	c.cancel()
}

func (task Task[I, O]) Prep(c *TasksCtx, input I) *TaskWithInput[I, O] {
	return &TaskWithInput[I, O]{c: c, task: task, input: input}
}

type AnyTaskWithInput interface {
	getTask() AnyTask
	getInput() any
	GetAny() (any, error)
}

type TaskWithInput[I any, O any] struct {
	c     *TasksCtx
	task  Task[I, O]
	input any
}

func (twi *TaskWithInput[I, O]) getTask() AnyTask { return twi.task }
func (twi *TaskWithInput[I, O]) getInput() any    { return twi.input }
func (twi TaskWithInput[I, O]) GetAny() (any, error) {
	return twi.task.Fn(&TasksCtxWithInput[I]{TasksCtx: twi.c, Input: twi.input.(I)})
}

func (twi TaskWithInput[I, O]) Get() (O, error) {
	return twi.task.Fn(&TasksCtxWithInput[I]{TasksCtx: twi.c, Input: twi.input.(I)})
}

type AnyTaskWithInputImpl struct {
	c     *TasksCtx
	task  AnyTask
	input any
}

func (twi *AnyTaskWithInputImpl) getTask() AnyTask { return twi.task }
func (twi *AnyTaskWithInputImpl) getInput() any    { return twi.input }

func (twi AnyTaskWithInputImpl) GetAny() (any, error) {
	twi.c.ParallelPreload(PrepAny(twi.c, twi.task, twi.input))
	x := twi.c.results.results[twi.task.getID()]
	return x.Data, x.Err
}

func PrepAny[I any](c *TasksCtx, task AnyTask, input I) AnyTaskWithInput {
	return &AnyTaskWithInputImpl{c: c, task: task, input: input}
}

func (c *TasksCtx) ParallelPreload(tasksWithInput ...AnyTaskWithInput) bool {
	if len(tasksWithInput) == 0 {
		return true
	}

	if len(tasksWithInput) == 1 {
		t := tasksWithInput[0]
		c.doOnce(t.getTask().getID(), c, t.getInput())
		return c.results.AllOK()
	}

	var wg sync.WaitGroup
	wg.Add(len(tasksWithInput))
	for _, t := range tasksWithInput {
		go func() {
			c.doOnce(t.getTask().getID(), c, t.getInput())
			wg.Done()
		}()
	}
	wg.Wait()

	return c.results.AllOK()
}

func (c *TasksCtx) doOnce(taskID int, ctx *TasksCtx, input any) {
	taskHelper := c.registry.registry[taskID]

	c.mu.Lock()
	if _, ok := c.results.results[taskID]; !ok {
		c.results.results[taskID] = &TaskResult{once: &sync.Once{}}
	}
	c.mu.Unlock()

	if c.context.Err() != nil {
		c.mu.Lock()
		c.results.results[taskID].Data = taskHelper.Phantom().OZero()
		c.results.results[taskID].Err = errors.New("parent context canceled before execution")
		c.mu.Unlock()
		return
	}

	c.getSyncOnce(taskID).Do(func() {
		// check if context is canceled
		if c.context.Err() != nil {
			c.mu.Lock()
			c.results.results[taskID].Data = taskHelper.Phantom().OZero()
			c.results.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
			return
		}

		resultChan := make(chan *TaskResult, 1)
		go func() {
			data, err := taskHelper.Execute(ctx, input)
			resultChan <- &TaskResult{Data: data, Err: err}
		}()

		select {
		case <-c.context.Done():
			c.mu.Lock()
			c.results.results[taskID].Data = taskHelper.Phantom().OZero()
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

func (c *TasksCtx) getSyncOnce(taskID int) *sync.Once {
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

type TaskResults struct {
	c       *TasksCtx
	results map[int]*TaskResult
}

func newResults(c *TasksCtx) *TaskResults {
	return &TaskResults{
		c:       c,
		results: make(map[int]*TaskResult),
	}
}

func (tr TaskResults) AllOK() bool {
	for _, result := range tr.results {
		if !result.OK() {
			return false
		}
	}
	return true
}

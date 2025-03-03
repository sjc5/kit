package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

type CtxMarker interface {
	getInput() any
	Params() Params
	SplatValues() []string
	TasksCtx() *tasks.TasksCtx
	Request() *http.Request
}

type RouterCtx[I any] struct {
	params      Params
	splatValues []string
	tasksCtx    *tasks.TasksCtx
	input       I
}

// implements CtxMarker
func (c *RouterCtx[I]) getInput() any             { return c.input }
func (c *RouterCtx[I]) Params() Params            { return c.params }
func (c *RouterCtx[I]) SplatValues() []string     { return c.splatValues }
func (c *RouterCtx[I]) TasksCtx() *tasks.TasksCtx { return c.tasksCtx }
func (c *RouterCtx[I]) Request() *http.Request    { return c.tasksCtx.Request() }

// does not implement CtxMarker
func (c *RouterCtx[I]) Input() I { return c.input }

/////////////////////////////////////////////////////////////////////
/////// CTX GETTERS
/////////////////////////////////////////////////////////////////////

type ctxGetter interface {
	getCtx(r *http.Request, match *matcher.Match) CtxMarker
}

type ctxGetterImpl[I any] func(*http.Request, *matcher.Match) *RouterCtx[I]

// implements ctxGetter
func (f ctxGetterImpl[I]) getCtx(r *http.Request, m *matcher.Match) CtxMarker {
	return f(r, m)
}

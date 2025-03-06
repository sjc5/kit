package mux

import (
	"github.com/sjc5/kit/pkg/genericsutil"
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/tasks"
)

type (
	None                      = genericsutil.None
	TaskHandler[I any, O any] = tasks.RegisteredTask[*ReqData[I], O]
	Params                    = matcher.Params
)

type ReqData[I any] struct {
	_params     Params
	_splat_vals []string
	_tasks_ctx  *tasks.TasksCtx
	_input      I
}

func (rd *ReqData[I]) Input() I { return rd._input }

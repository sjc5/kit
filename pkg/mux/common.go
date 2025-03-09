package mux

import (
	"github.com/sjc5/kit/pkg/genericsutil"
	"github.com/sjc5/kit/pkg/matcher"
	"github.com/sjc5/kit/pkg/response"
	"github.com/sjc5/kit/pkg/tasks"
)

type (
	None                      = genericsutil.None
	TaskHandler[I any, O any] = tasks.RegisteredTask[*ReqData[I], O]
	Params                    = matcher.Params
)

type ReqData[I any] struct {
	_params         Params
	_splat_vals     []string
	_tasks_ctx      *tasks.TasksCtx
	_input          I
	_response_proxy *response.Proxy
}

func NewReqDataFromExistingWithFreshResponseProxy[I any](rd _Req_Data_Marker) _Req_Data_Marker {
	return &ReqData[I]{
		_params:         rd._params,
		_splat_vals:     rd._splat_vals,
		_tasks_ctx:      rd._tasks_ctx,
		_input:          rd._input,
		_response_proxy: response.NewProxy(),
	}
}

package nestedrouter

import "github.com/sjc5/kit/pkg/tasks"

type Task = tasks.Task
type Loader[O any] func(*Ctx) (O, error)
type Action[I any, O any] func(*Ctx, I) (O, error)

type DataFunctionMarker interface {
	GetInputZeroValue() any
	GetOutputZeroValue() any
}

func (Loader[I]) GetInputZeroValue() any {
	var zero I
	return zero
}
func (Loader[O]) GetOutputZeroValue() any {
	var zero O
	return zero
}
func (Action[I, O]) GetInputZeroValue() any {
	var zero I
	return zero
}
func (Action[I, O]) GetOutputZeroValue() any {
	var zero O
	return zero
}

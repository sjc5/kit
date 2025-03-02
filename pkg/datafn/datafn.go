package datafn

// UnwrappedAny interface for type-erased functions
type UnwrappedAny interface {
	GetInputZeroValue() any
	GetOutputZeroValue() any
}

// Single-argument version
type Unwrapped[I any, O any] func(I) (O, error)

func (u Unwrapped[I, O]) GetInputZeroValue() any {
	var i I
	return i
}

func (u Unwrapped[I, O]) GetOutputZeroValue() any {
	var o O
	return o
}

// Two-argument version
type Unwrapped2[C, I, O any] func(C, I) (O, error)

func (u Unwrapped2[C, I, O]) GetInputZeroValue() any {
	var i I
	return i
}

func (u Unwrapped2[C, I, O]) GetOutputZeroValue() any {
	var o O
	return o
}

func (u Unwrapped2[C, I, O]) GetCtxZeroValue() any {
	var c C
	return c
}

// Type-erased interface with Execute method
type WrappedAny interface {
	UnwrappedAny
	Execute(any) (any, error)
}

// Two-argument type-erased interface
type WrappedAny2 interface {
	UnwrappedAny
	Execute2(any, any) (any, error)
}

// Single-argument wrapped function
type Wrapped[I any, O any] struct {
	U Unwrapped[I, O]
}

func (w Wrapped[I, O]) GetInputZeroValue() any {
	return w.U.GetInputZeroValue()
}

func (w Wrapped[I, O]) GetOutputZeroValue() any {
	return w.U.GetOutputZeroValue()
}

func (w Wrapped[I, O]) Execute(i any) (any, error) {
	_, ok := i.(I)
	if !ok {
		var zero I
		return w.U(zero)
	}
	return w.U(i.(I))
}

// Constructor for Wrapped
func NewWrapped[I any, O any](u Unwrapped[I, O]) Wrapped[I, O] {
	return Wrapped[I, O]{U: u}
}

// Two-argument wrapped function
type Wrapped2[C, I, O any] struct {
	U Unwrapped2[C, I, O]
}

func (w Wrapped2[C, I, O]) GetInputZeroValue() any {
	return w.U.GetInputZeroValue()
}

func (w Wrapped2[C, I, O]) GetOutputZeroValue() any {
	return w.U.GetOutputZeroValue()
}

func (w Wrapped2[C, I, O]) Execute2(c any, i any) (any, error) {
	_, ok := i.(I)
	if !ok {
		var zero I
		return w.U(c.(C), zero)
	}
	return w.U(c.(C), i.(I))
}

// Constructor for Wrapped2
func NewWrapped2[C, I, O any](u Unwrapped2[C, I, O]) Wrapped2[C, I, O] {
	return Wrapped2[C, I, O]{U: u}
}

// Adapter to convert Unwrapped2 to Unwrapped
func ToUnwrapped[C, I, O any](u Unwrapped2[C, I, O], ctx C) Unwrapped[I, O] {
	return func(i I) (O, error) {
		return u(ctx, i)
	}
}

// Adapter to convert Unwrapped to Unwrapped2
func ToUnwrapped2[C, I, O any](u Unwrapped[I, O]) Unwrapped2[C, I, O] {
	return func(c C, i I) (O, error) {
		return u(i)
	}
}

// Package datafn provides helpers for implementing type erasure patterns
package datafn

type Any interface {
	Phantom() _phantom
}

type _phantom interface {
	IZero() any   // returns I zero val
	OZero() any   // returns O zero val
	NewIPtr() any // calls `new(I)` and returns the pointer to I
	NewOPtr() any // calls `new(O)` and returns the pointer to O
}

type PhantomImpl[I any, O any] struct{}

func (_ *PhantomImpl[I, O]) IZero() any {
	var zero I
	return zero
}
func (_ *PhantomImpl[I, O]) OZero() any {
	var zero O
	return zero
}
func (_ *PhantomImpl[I, O]) NewIPtr() any       { return new(I) }
func (_ *PhantomImpl[I, O]) NewOPtr() any       { return new(O) }
func (pi *PhantomImpl[I, O]) Phantom() _phantom { return pi }

func ToPhantom[I any, O any]() *PhantomImpl[I, O] {
	return &PhantomImpl[I, O]{}
}

/////////////////////////////////////////////////////////////////////
/////// ONE-ARG FUNCTIONS ("Fn")
/////////////////////////////////////////////////////////////////////

/////// UNWRAPPED

type Fn[I any, O any] func(I) (O, error)

func (_ Fn[I, O]) Phantom() _phantom { return ToPhantom[I, O]() }

func ToPhantomFn[I any, O any]() Fn[I, O] {
	return func(_ I) (O, error) {
		var zero O
		return zero, nil
	}
}

/////// WRAPPED

type AnyFnWrapped interface {
	Any
	Execute(any) (any, error)
}

type FnWrapped[I any, O any] struct {
	Fn Fn[I, O]
}

func (_ FnWrapped[I, O]) Phantom() _phantom { return ToPhantom[I, O]() }

func (w FnWrapped[I, O]) Execute(i any) (any, error) {
	_, ok := i.(I)
	if !ok {
		var zero I
		return w.Fn(zero)
	}
	return w.Fn(i.(I))
}

/////// WRAPPING HELPER

func FnToWrapped[I any, O any](u Fn[I, O]) FnWrapped[I, O] {
	return FnWrapped[I, O]{Fn: u}
}

/////////////////////////////////////////////////////////////////////
/////// TWO-ARG FUNCTIONS ("CtxFn")
/////////////////////////////////////////////////////////////////////

/////// UNWRAPPED

type CtxFn[Ctx, I, O any] func(Ctx, I) (O, error)

func (_ CtxFn[Ctx, I, O]) Phantom() _phantom { return ToPhantom[I, O]() }

/////// WRAPPED

type AnyCtxFnWrapped interface {
	Any
	Execute(any, any) (any, error)
}

type CtxFnWrapped[Ctx, I, O any] struct {
	CtxFn CtxFn[Ctx, I, O]
}

func (_ CtxFnWrapped[Ctx, I, O]) Phantom() _phantom { return ToPhantom[I, O]() }

func (w CtxFnWrapped[Ctx, I, O]) Execute(c any, i any) (any, error) {
	_, ok := i.(I)
	if !ok {
		var zero I
		return w.CtxFn(c.(Ctx), zero)
	}
	return w.CtxFn(c.(Ctx), i.(I))
}

/////// WRAPPING HELPER

func CtxFnToWrapped[Ctx, I, O any](u CtxFn[Ctx, I, O]) CtxFnWrapped[Ctx, I, O] {
	return CtxFnWrapped[Ctx, I, O]{CtxFn: u}
}

/////////////////////////////////////////////////////////////////////
/////// Fn / CtxFn ADAPTERS
/////////////////////////////////////////////////////////////////////

func CtxFnToFn[Ctx, I, O any](u CtxFn[Ctx, I, O], ctx Ctx) Fn[I, O] {
	return func(i I) (O, error) {
		return u(ctx, i)
	}
}

func FnToCtxFn[Ctx, I, O any](u Fn[I, O]) CtxFn[Ctx, I, O] {
	return func(c Ctx, i I) (O, error) {
		return u(i)
	}
}

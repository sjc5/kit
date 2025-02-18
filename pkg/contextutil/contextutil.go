package contextutil

import (
	"context"
	"net/http"
)

type Store[T any] struct {
	key keyWrapper
}

type keyWrapper struct {
	name string
}

func NewStore[T any](key string) *Store[T] {
	return &Store[T]{key: keyWrapper{name: key}}
}

func (h *Store[T]) GetContextWithValue(r *http.Request, val T) context.Context {
	return context.WithValue(r.Context(), h.key, val)
}

func (h *Store[T]) GetValueFromContext(r *http.Request) T {
	ctx := r.Context()
	val := ctx.Value(h.key)
	if val == nil {
		var zeroVal T
		return zeroVal
	}
	return val.(T)
}

func (h *Store[T]) GetRequestWithContext(r *http.Request, val T) *http.Request {
	return r.WithContext(h.GetContextWithValue(r, val))
}

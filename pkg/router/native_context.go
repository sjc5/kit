package router

import (
	"context"
	"net/http"
)

type nativeContextKey string

const routerCtxNativeContextKey nativeContextKey = "routerCtx"

func addRouterCtxToNativeContext(r *http.Request, c CtxMarker) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), routerCtxNativeContextKey, c),
	)
}

func getRouterCtxFromNativeContext[I any](r *http.Request) *RouterCtx[I] {
	if ctx, ok := r.Context().Value(routerCtxNativeContextKey).(*RouterCtx[I]); ok {
		return ctx
	}
	return nil
}

func getParamFromNativeContext[I any](r *http.Request, name string) string {
	return getParamsFromNativeContext[I](r)[name]
}

func getParamsFromNativeContext[I any](r *http.Request) Params {
	if routerCtx := getRouterCtxFromNativeContext[I](r); routerCtx != nil {
		return routerCtx.params
	}
	return nil
}

func getSplatValuesFromNativeContext[I any](r *http.Request) []string {
	if routerCtx := getRouterCtxFromNativeContext[I](r); routerCtx != nil {
		return routerCtx.splatValues
	}
	return nil
}

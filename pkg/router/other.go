package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
)

type method = string
type pattern = string

type Params = matcher.Params
type ClassicHandler = http.Handler
type ClassicHandlerFunc = http.HandlerFunc
type ClassicMiddleware func(ClassicHandler) ClassicHandler

var permittedHTTPMethods = map[method]struct{}{
	// query
	http.MethodGet:  {},
	http.MethodHead: {},

	// mutation
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},

	// other
	http.MethodConnect: {},
	http.MethodOptions: {},
	http.MethodTrace:   {},
}

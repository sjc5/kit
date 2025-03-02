package router

import (
	"net/http"

	"github.com/sjc5/kit/pkg/matcher"
)

type Params = matcher.Params
type Method = string
type Pattern = string
type StdHandler = http.Handler
type StdHandlerFunc = http.HandlerFunc
type StdMiddleware func(StdHandler) StdHandler

var permittedHTTPMethods = map[Method]struct{}{
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

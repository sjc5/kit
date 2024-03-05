package chirpc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Def struct {
	Name        string
	Endpoint    string
	Type        Type
	Input       interface{}
	Output      interface{}
	HandlerFunc http.HandlerFunc
	Procedure   string
}

type Type string
type Procedure string

const (
	TypeQuery    Type = "query"
	TypeMutation Type = "mutation"
)

func (d *Def) Register(r chi.Router) {
	method := "GET"
	if d.Type == TypeMutation {
		method = "POST"
	}
	r.Method(method, d.Endpoint, d.HandlerFunc)
}

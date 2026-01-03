package routes

import (
	"fmt"
	"net/http"
)

type Route struct {
	method  string
	path    string
	handler http.HandlerFunc
}

func NewRoute(method string, path string, handler http.HandlerFunc) *Route {
	return &Route{
		method:  method,
		path:    path,
		handler: handler,
	}
}

type Middeware func(http.HandlerFunc) http.HandlerFunc

func (r *Route) WithMiddleware(mws ...Middeware) *Route {
	for _, mw := range mws {
		r.handler = mw(r.handler)
	}
	return r
}

func (r *Route) Register(mux *http.ServeMux) {
	mux.HandleFunc(fmt.Sprintf("%s %s", r.method, r.path), r.handler)
}

func (r *Route) Method() string {
	return r.method
}

func (r *Route) Path() string {
	return r.path
}

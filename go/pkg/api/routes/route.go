package routes

import (
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Handler[TRequest any, TResponse any] func(session.Session[TRequest, TResponse]) error

type Route[TRequest any, TResponse any] struct {
	method  string
	path    string
	handler func(session.Session[TRequest, TResponse]) error
}

func NewRoute[TRequest any, TResponse any](method string, path string, handler Handler[TRequest, TResponse]) *Route[TRequest, TResponse] {
	return &Route[TRequest, TResponse]{
		method:  method,
		path:    path,
		handler: handler,
	}
}

type Middeware[TRequest any, TResponse any] func(Handler[TRequest, TResponse]) Handler[TRequest, TResponse]

func (r *Route[TRequest, TResponse]) WithMiddleware(mws ...Middeware[TRequest, TResponse]) *Route[TRequest, TResponse] {
	for _, mw := range mws {
		r.handler = mw(r.handler)
	}
	return r
}

func (r *Route[TRequest, TResponse]) Handle(sess session.Session[TRequest, TResponse]) error {
	return r.handler(sess)
}

func (r *Route[TRequest, TResponse]) Method() string {
	return r.method
}

func (r *Route[TRequest, TResponse]) Path() string {
	return r.path
}

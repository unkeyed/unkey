package routes

import (
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Route[TRequest session.Redacter, TResponse session.Redacter] struct {
	method   string
	path     string
	handleFn func(session.Session[TRequest, TResponse]) error
}

func NewRoute[TRequest session.Redacter, TResponse session.Redacter](method string, path string, handleFn func(session.Session[TRequest, TResponse]) error) Route[TRequest, TResponse] {
	return Route[TRequest, TResponse]{
		method:   method,
		path:     path,
		handleFn: handleFn,
	}
}

type Middeware[TRequest session.Redacter, TResponse session.Redacter] func(session.Handler[TRequest, TResponse]) (session.Handler[TRequest, TResponse], error)

func (r *Route[TRequest, TResponse]) WithMiddleware(mws ...Middeware[TRequest, TResponse]) *Route[TRequest, TResponse] {

	for _, mw := range mws {
		var err error
		next, err := mw(r)

		if err != nil {
			panic("Middleware Error" + err.Error())
		}
		r.handleFn = next.Handle
	}
	return r
}

func (r *Route[TRequest, TResponse]) Handle(sess session.Session[TRequest, TResponse]) error {
	return r.handleFn(sess)
}

func (r *Route[TRequest, TResponse]) Method() string {
	return r.method
}

func (r *Route[TRequest, TResponse]) Path() string {
	return r.path
}

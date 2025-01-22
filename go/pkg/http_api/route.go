package httpApi

type Route[TRequest Redacter, TResponse Redacter] interface {
	Method() string
	Path() string
	Handle(sess *Session[TRequest, TResponse]) error
	WithMiddleware(...Middleware[TRequest, TResponse]) Route[TRequest, TResponse]
}

type route[TRequest Redacter, TResponse Redacter] struct {
	method   string
	path     string
	handleFn HandleFunc[TRequest, TResponse]
}

func NewRoute[TRequest Redacter, TResponse Redacter](method string, path string, handleFn func(*Session[TRequest, TResponse]) error) Route[TRequest, TResponse] {
	return &route[TRequest, TResponse]{
		method:   method,
		path:     path,
		handleFn: handleFn,
	}
}

func (r *route[TRequest, TResponse]) WithMiddleware(mws ...Middleware[TRequest, TResponse]) Route[TRequest, TResponse] {

	for _, mw := range mws {
		r.handleFn = mw.Handle(r.handleFn)
	}
	return r
}

func (r *route[TRequest, TResponse]) Handle(sess *Session[TRequest, TResponse]) error {
	return r.handleFn(sess)
}

func (r *route[TRequest, TResponse]) Method() string {
	return r.method
}

func (r *route[TRequest, TResponse]) Path() string {
	return r.path
}

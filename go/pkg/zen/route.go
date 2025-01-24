package zen

type Route interface {
	Method() string
	Path() string
	Handle(sess *Session) error
	WithMiddleware(...Middleware) Route
}

type route struct {
	method   string
	path     string
	handleFn HandleFunc
}

func NewRoute(method string, path string, handleFn func(*Session) error) *route {
	return &route{
		method:   method,
		path:     path,
		handleFn: handleFn,
	}
}

func (r *route) WithMiddleware(mws ...Middleware) Route {
	for _, mw := range mws {
		r.handleFn = mw(r.handleFn)
	}
	return r
}

func (r *route) Handle(sess *Session) error {
	return r.handleFn(sess)
}

func (r *route) Method() string {
	return r.method
}

func (r *route) Path() string {
	return r.path
}

package zen

import "context"

// Route represents an HTTP endpoint with its method, path, and handler function.
// It encapsulates the behavior of a specific HTTP endpoint in the system.
type Route interface {
	// Method returns the HTTP method this route responds to (GET, POST, etc.).
	Method() string

	// Path returns the URL path pattern this route matches.
	Path() string

	// Handle processes the HTTP request encapsulated by the Session.
	// It should return an error if processing fails, which will then be
	// handled by error middleware.
	Handle(context.Context, *Session) error

	// WithMiddleware returns a new Route with the provided middleware applied.
	// Middleware is applied in the order provided, with each wrapping the next.
	WithMiddleware(...Middleware) Route
}

type route struct {
	method   string
	path     string
	handleFn HandleFunc
}

// NewRoute creates a standard Route implementation with the specified method,
// path, and handler function.
//
// Example:
//
//	route := zen.NewRoute("POST", "/api/users", func(s *zen.Session) error {
//	    var user User
//	    if err := s.BindBody(&user); err != nil {
//	        return err
//	    }
//	    result, err := createUser(s.Context(), user)
//	    if err != nil {
//	        return err
//	    }
//	    return s.JSON(http.StatusCreated, result)
//	})
func NewRoute(method string, path string, handleFn func(context.Context, *Session) error) *route {
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

func (r *route) Handle(ctx context.Context, sess *Session) error {
	return r.handleFn(ctx, sess)
}

func (r *route) Method() string {
	return r.method
}

func (r *route) Path() string {
	return r.path
}

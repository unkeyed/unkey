package zen

import "context"

// CATCHALL is a special method constant that indicates a route should handle all HTTP methods.
// When a route returns CATCHALL (empty string) from Method(), it will be registered without
// a method prefix, allowing it to match all HTTP methods.
const CATCHALL = ""

// Route represents an HTTP endpoint with its method, path, and handler function.
// It encapsulates the behavior of a specific HTTP endpoint in the system.
type Route interface {
	Handler

	// Method returns the HTTP method this route responds to (GET, POST, etc.).
	// Return CATCHALL to handle all HTTP methods.
	Method() string

	// Path returns the URL path pattern this route matches.
	Path() string
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
//	route := zen.NewRoute("POST", "/api/users", func(ctx context.Context, s *zen.Session) error {
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

func (r *route) Method() string {
	return r.method
}

func (r *route) Path() string {
	return r.path
}

func (r *route) Handle(ctx context.Context, sess *Session) error {
	return r.handleFn(ctx, sess)
}

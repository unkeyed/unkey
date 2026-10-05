package zen

import (
	"bytes"
	"context"
	"net/http"
)

// RoutingError represents a mux-generated 404 or 405, not a registered handler's response.
type RoutingError struct {
	Status int
}

func (e *RoutingError) Error() string {
	return http.StatusText(e.Status)
}

// RegisterRoutingErrors sends mux-generated errors through middlewares.
// Call before serving requests. A nil chain keeps Go's default responses.
// Request bodies are not read, and the mux's Allow header is preserved.
func (s *Server) RegisterRoutingErrors(middlewares []Middleware) {
	s.routingErrorMiddlewares = middlewares
}

// ServeHTTP dispatches requests through the mux and optional routing error middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(s.routingErrorMiddlewares) == 0 || r.RequestURI == "*" {
		s.mux.ServeHTTP(w, r)
		return
	}

	handler, pattern := s.mux.Handler(r)
	if pattern != "" {
		s.mux.ServeHTTP(w, r)
		return
	}

	response := routingResponse{
		Buffer: bytes.Buffer{},
		header: w.Header().Clone(),
		status: http.StatusOK,
	}
	handler.ServeHTTP(&response, r)
	for name, values := range response.header {
		w.Header()[name] = values
	}

	s.handler(s.routingErrorMiddlewares, func(_ context.Context, sess *Session) error {
		if response.status == http.StatusNotFound || response.status == http.StatusMethodNotAllowed {
			w.Header().Del("Content-Type")
			w.Header().Del("Content-Length")
			return &RoutingError{Status: response.status}
		}
		return sess.Send(response.status, response.Bytes())
	}, true).ServeHTTP(w, r)
}

type routingResponse struct {
	bytes.Buffer
	header http.Header
	status int
}

func (r *routingResponse) Header() http.Header {
	return r.header
}

func (r *routingResponse) WriteHeader(status int) {
	r.status = status
}

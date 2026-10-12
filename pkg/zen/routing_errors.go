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
	if len(middlewares) == 0 {
		s.routingErrors = nil
		return
	}
	s.routingErrors = s.handler(middlewares, s.routingError, true)
}

// ServeHTTP dispatches requests through the mux and optional routing error middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.routingErrors == nil || r.RequestURI == "*" {
		s.mux.ServeHTTP(w, r)
		return
	}

	_, pattern := s.mux.Handler(r)
	if pattern != "" {
		s.mux.ServeHTTP(w, r)
		return
	}
	s.routingErrors.ServeHTTP(w, r)
}

func (s *Server) routingError(_ context.Context, sess *Session) error {
	handler, _ := s.mux.Handler(sess.Request())
	header := sess.ResponseWriter().Header()
	response := routingResponse{
		Buffer: bytes.Buffer{},
		header: header.Clone(),
		status: http.StatusOK,
	}
	handler.ServeHTTP(&response, sess.Request())
	for name, values := range response.header {
		header[name] = values
	}

	if response.status == http.StatusNotFound || response.status == http.StatusMethodNotAllowed {
		header.Del("Content-Type")
		header.Del("Content-Length")
		return &RoutingError{Status: response.status}
	}
	return sess.Send(response.status, response.Bytes())
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

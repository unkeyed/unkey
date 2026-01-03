package pprof

import (
	"context"
	"crypto/subtle"
	"net/http/pprof"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Handler handles pprof profiling endpoints
type Handler struct {
	Logger   logging.Logger
	Username string
	Password string
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "GET"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/debug/pprof/{path...}"
}

// Handle processes the HTTP request and delegates to pprof handlers
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Only require authentication if username and password are configured
	if h.Username != "" && h.Password != "" {
		username, password, ok := s.Request().BasicAuth()
		if !ok {
			s.ResponseWriter().Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			return fault.New("basic auth required",
				fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
				fault.Public("Basic authentication is required."))
		}

		// Check username and password with constant-time comparison
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(h.Username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(h.Password)) == 1

		if !usernameMatch || !passwordMatch {
			s.ResponseWriter().Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			return fault.New("invalid credentials",
				fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
				fault.Internal("pprof credentials do not match"),
				fault.Public("Invalid username or password."))
		}
	}

	// Get the pprof path from the wildcard match
	pprofPath := s.Request().PathValue("path")

	// Delegate to appropriate pprof handler based on path
	switch pprofPath {
	case "profile":
		pprof.Profile(s.ResponseWriter(), s.Request())
	case "cmdline":
		pprof.Cmdline(s.ResponseWriter(), s.Request())
	case "symbol":
		pprof.Symbol(s.ResponseWriter(), s.Request())
	case "trace":
		pprof.Trace(s.ResponseWriter(), s.Request())
	case "":
		pprof.Index(s.ResponseWriter(), s.Request())
	default:
		// For named profiles like heap, goroutine, threadcreate, etc.
		pprof.Handler(pprofPath).ServeHTTP(s.ResponseWriter(), s.Request())
	}

	return nil
}

package zen

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Server manages HTTP server configuration, route registration, and lifecycle.
// It provides connection pooling for session objects to reduce memory churn
// during request handling.
//
// Server instances should be created with the New function and can be safely
// used by multiple goroutines.
type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	mux         *http.ServeMux
	srv         *http.Server

	sessions sync.Pool
}

// Config configures the behavior of a Server instance.
type Config struct {
	// NodeID uniquely identifies this server instance, useful for logging and tracing.
	NodeID string

	// Logger provides structured logging for the server. If nil, logging is disabled.
	Logger logging.Logger
}

// New creates a new server with the provided configuration.
// It initializes the HTTP server and session pool with default timeouts.
//
// The HTTP server is configured with reasonable defaults for production use:
// - ReadTimeout: 10 seconds
// - WriteTimeout: 20 seconds
//
// Example:
//
//	server, err := zen.New(zen.Config{
//	    NodeID: "api-server-1",
//	    Logger: logger,
//	})
//	if err != nil {
//	    log.Fatalf("failed to initialize server: %v", err)
//	}
func New(config Config) (*Server, error) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		// See https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		//
		// > # http.ListenAndServe is doing it wrong
		// > Incidentally, this means that the package-level convenience functions that bypass http.Server
		// > like http.ListenAndServe, http.ListenAndServeTLS and http.Serve are unfit for public Internet
		// > Servers.
		// >
		// > Those functions leave the Timeouts to their default off value, with no way of enabling them,
		// > so if you use them you'll soon be leaking connections and run out of file descriptors. I've
		// > made this mistake at least half a dozen times.
		// >
		// > Instead, create a http.Server instance with ReadTimeout and WriteTimeout and use its
		// > corresponding methods, like in the example a few paragraphs above.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	s := &Server{
		mu:          sync.Mutex{},
		logger:      config.Logger,
		isListening: false,
		mux:         mux,
		srv:         srv,
		sessions: sync.Pool{
			New: func() any {
				return &Session{
					workspaceID:    "",
					requestID:      "",
					w:              nil,
					r:              nil,
					requestBody:    []byte{},
					responseStatus: 0,
					responseBody:   []byte{},
				}
			},
		},
	}
	return s, nil
}

// Get a fresh or reused session from the pool.
//
// You must return it to the pool when you are done via s.returnSession()
//
// Returning proper types is a bit annoying here, cause each session might have
// different requets and response types, so we just return any.
// You should immediately cast it to your desired type
//
// sess := s.getSession().(session.Session[MyRequest, MyResponse]).
func (s *Server) getSession() any {
	return s.sessions.Get()
}

// Return the session to the sync pool.
func (s *Server) returnSession(session any) {
	s.sessions.Put(session)
}

// Mux returns the underlying http.ServeMux.
// This is primarily intended for testing and advanced usage scenarios.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// Listen starts the HTTP server on the specified address.
// This method blocks until the server shuts down or encounters an error.
// Once listening, the server will not start again if Listen is called multiple times.
//
// Example:
//
//	// Start server in a goroutine to allow for graceful shutdown
//	go func() {
//	    if err := server.Listen(ctx, ":8080"); err != nil {
//	        log.Printf("server stopped: %v", err)
//	    }
//	}()
func (s *Server) Listen(ctx context.Context, addr string) error {
	s.mu.Lock()
	if s.isListening {
		s.logger.Warn("already listening")
		s.mu.Unlock()
		return nil
	}
	s.isListening = true
	s.mu.Unlock()

	s.srv.Addr = addr

	s.logger.Info("listening", "addr", addr)

	err := s.srv.ListenAndServe()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("listening failed", ""))
	}
	return nil
}

// RegisterRoute adds an HTTP route to the server with the specified middleware chain.
// Routes are matched by both method and path.
//
// Middleware is applied in the order provided, with each middleware wrapping the next.
// The innermost handler (last to execute) is the route's handler.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithLogging(logger), zen.WithErrorHandling()},
//	    zen.NewRoute("GET", "/health", healthCheckHandler),
//	)
func (s *Server) RegisterRoute(middlewares []Middleware, route Route) {
	s.logger.Info("registering",
		"method", route.Method(),
		"path", route.Path(),
	)
	s.mux.HandleFunc(
		fmt.Sprintf("%s %s", route.Method(), route.Path()),
		func(w http.ResponseWriter, r *http.Request) {
			sess, ok := s.getSession().(*Session)
			if !ok {
				panic("Unable to cast session")
			}
			defer func() {
				sess.reset()
				s.returnSession(sess)
			}()

			err := sess.init(w, r)
			if err != nil {
				s.logger.Error("failed to init session")
				return
			}

			// Apply middleware
			var handle HandleFunc = route.Handle

			// Reverses the middlewares to run in the desired order.
			// If middlewares are [A, B, C], this writes [C, B, A] to s.middlewares.
			for i := len(middlewares) - 1; i >= 0; i-- {
				handle = middlewares[i](handle)
			}

			err = handle(r.Context(), sess)

			if err != nil {
				panic(err)
			}
		})
}

// Shutdown gracefully stops the HTTP server, allowing in-flight requests
// to complete before returning.
//
// Example:
//
//	// Handle shutdown signal
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := server.Shutdown(ctx); err != nil {
//	    log.Printf("server shutdown error: %v", err)
//	}
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.srv.Close()
	if err != nil {
		return fault.Wrap(err)
	}
	return nil
}

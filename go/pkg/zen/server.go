package zen

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	mux         *http.ServeMux
	srv         *http.Server

	// middlewares in the order of inner -> outer
	// The last middleware in this slice will run first
	middlewares []Middleware
	sessions    sync.Pool
}

type Config struct {
	NodeId     string
	Logger     logging.Logger
	Clickhouse EventBuffer
}

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
					requestID:      "",
					w:              nil,
					r:              nil,
					requestBody:    []byte{},
					responseStatus: 0,
					responseBody:   []byte{},
				}
			},
		},
		middlewares: []Middleware{},
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

// Calling this function multiple times will have no effect.
func (s *Server) Listen(ctx context.Context, addr string) error {
	s.mu.Lock()
	if s.isListening {
		s.logger.Warn(ctx, "already listening")
		s.mu.Unlock()
		return nil
	}
	s.isListening = true
	s.mu.Unlock()

	s.srv.Addr = addr

	s.logger.Info(ctx, "listening", slog.String("addr", addr))
	err := s.srv.ListenAndServe()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("listening failed", ""))
	}
	return nil
}

// SetGlobalMiddleware sets middleware that will be executed before every
// route handler.
// Global middleware are executed in the order they are added, before any
// route-specific middleware.
// Each middleware can execute code before and after the handler it wraps.
//
// Request/Response flow:
// SetGlobalMiddleware(Middleware_1, Middleware_2)
//
//	                        │ REQUEST
//		                      ▼
//		┌──────────── Global Middleware 1 ────────────┐
//		│                     │                       │
//		│                     ▼                       │
//		│      ┌────── Global Middleware 2 ─────┐     │
//		│      │              │                 │     │
//		│      │              ▼                 │     │
//		│      │      ┌─── Route MW ────┐       │     │
//		│      │      │       │         │       │     │
//		│      │      │       ▼         │       │     │
//		│      │      │   ┌──────────┐  │       │     │
//		│      │      │   │ Handler  │  │       │     │
//		│      │      │   └──────────┘  │       │     │
//		│      │      │       │         │       │     │
//		│      │      │       ▼         │       │     │
//		│      │      └─────────────────┘       │     │
//		│      │              │                 │     │
//		│      │              ▼                 │     │
//		│      └────────────────────────────────┘     │
//		│                     │                       │
//		│                     ▼                       │
//		└─────────────────────────────────────────────┘
//		                      │
//		                      ▼ RESPONSE
//
// Example usage:
//
//	router.SetGlobalMiddleware(
//	    logging.Middleware,    // logs all requests
//	    auth.ValidateToken,    // validates auth tokens
//	)
//
// Note: Global middleware cannot be removed once set. To modify global middleware,
// you need to set all desired middleware again with a new call to SetGlobalMiddleware.
func (s *Server) SetGlobalMiddleware(middlewares ...Middleware) {
	n := len(middlewares)
	if n == 0 {
		return
	}

	// Reverses the middlewares to run in the desired order.
	// If middlewares are [A, B, C], this writes [C, B, A] to s.middlewares.
	s.middlewares = make([]Middleware, n)
	for i, mw := range middlewares {
		s.middlewares[n-i-1] = mw
	}
}
func (s *Server) RegisterRoute(route Route) {
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

			err := sess.Init(w, r)
			if err != nil {
				s.logger.Error(context.Background(), "failed to init session")
				return
			}

			// Apply middleware
			var handle HandleFunc = route.Handle

			for _, mw := range s.middlewares {
				handle = mw(handle)
			}

			err = handle(sess)

			if err != nil {
				panic(err)
			}
		})
}

func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.srv.Close()
	if err != nil {
		return fault.Wrap(err)
	}
	return nil
}

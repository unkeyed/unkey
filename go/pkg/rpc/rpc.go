package rpc

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	srv         *http.Server
	mux         *http.ServeMux
	// Define fields for the server
}

type Config struct {
	Logger logging.Logger

	RatelimitService ratelimitv1connect.RatelimitServiceHandler
	// Define fields for the configuration
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

	interceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()), otelconnect.WithTrustRemote(),
	)
	if err != nil {
		return nil, err
	}

	if config.RatelimitService != nil {
		mux.Handle(
			ratelimitv1connect.NewRatelimitServiceHandler(
				config.RatelimitService,
				connect.WithInterceptors(interceptor),
			),
		)
	}

	return &Server{
		mu:          sync.Mutex{},
		logger:      config.Logger,
		isListening: false,
		srv:         srv,
		mux:         mux,
	}, nil
}

// Listen starts the RPC server on the specified address.
// This method blocks until the server shuts down or encounters an error.
// Once listening, the server will not start again if Listen is called multiple times.
//
// Example:
//
//	// Start server in a goroutine to allow for graceful shutdown
//	go func() {
//	    if err := server.Listen(ctx, ":8080"); err != nil {
//	        log.Printf("rpc stopped: %v", err)
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

	s.logger.Info("listening", "srv", "rpc", "addr", addr)

	err := s.srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fault.Wrap(err, fault.WithDesc("listening failed", ""))
	}
	return nil
}

// Shutdown gracefully stops the RPC server, allowing in-flight requests
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

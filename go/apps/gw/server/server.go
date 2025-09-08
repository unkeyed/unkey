package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Server manages the gateway HTTP server with TLS termination support.
// It acts as a reverse proxy, forwarding requests to backend services.
type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	srv         *http.Server
	handler     http.Handler
	certManager CertificateManager
	enableTLS   bool

	sessions sync.Pool
}

// CertificateManager handles dynamic certificate selection for multiple domains.
type CertificateManager interface {
	// GetCertificate returns the certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

// Config configures the gateway server.
type Config struct {
	// Logger provides structured logging for the server.
	Logger logging.Logger

	// Handler is the HTTP handler for processing requests.
	// This will typically be wrapped with middleware.
	Handler http.Handler

	// CertManager handles dynamic TLS certificate selection.
	// If provided, the server will handle TLS termination with SNI support.
	CertManager CertificateManager

	// EnableTLS enables HTTPS mode. If true, CertManager must be provided.
	EnableTLS bool
}

// New creates a new gateway server with the provided configuration.
func New(config Config) (*Server, error) {
	if config.EnableTLS && config.CertManager == nil {
		return nil, fmt.Errorf("certificate manager required when TLS is enabled")
	}

	srv := &http.Server{
		Handler:      config.Handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Configure TLS if enabled
	if config.EnableTLS {
		config.Logger.Info("Configuring TLS")

		srv.TLSConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return config.CertManager.GetCertificate(context.Background(), hello.ServerName)
			},
			MinVersion: tls.VersionTLS13,
		}
	}

	s := &Server{
		mu:          sync.Mutex{},
		logger:      config.Logger,
		isListening: false,
		srv:         srv,
		handler:     config.Handler,
		certManager: config.CertManager,
		enableTLS:   config.EnableTLS,
		sessions: sync.Pool{
			New: func() any {
				return &Session{
					WorkspaceID:    "",
					requestID:      "",
					startTime:      time.Time{},
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

// Serve starts the HTTP server on the provided listener.
// This method blocks until the server shuts down or encounters an error.
// If EnableTLS is true, the server will handle TLS termination with SNI support.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.mu.Lock()
	if s.isListening {
		s.logger.Warn("already listening")
		s.mu.Unlock()
		return nil
	}

	s.isListening = true
	s.mu.Unlock()

	// Set up context handling for graceful shutdown
	serverCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle context cancellation in a separate goroutine
	go func() {
		select {
		case <-ctx.Done():
			s.logger.Info("shutting down server due to context cancellation")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			err := s.Shutdown(shutdownCtx)
			if err != nil {
				s.logger.Error("error during server shutdown", "error", err.Error())
			}
		case <-serverCtx.Done():
			// Server stopped on its own
		}
	}()

	var err error

	// Check if TLS should be used
	if s.enableTLS {
		s.logger.Info("listening with TLS termination (SNI enabled)", "addr", ln.Addr().String())
		err = s.srv.ServeTLS(ln, "", "")
	} else {
		s.logger.Info("listening without TLS", "addr", ln.Addr().String())
		err = s.srv.Serve(ln)
	}

	// Cancel the server context since the server has stopped
	cancel()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fault.Wrap(err, fault.Internal("listening failed"))
	}
	return nil
}

// getSession gets a session from the pool.
func (s *Server) getSession() any {
	return s.sessions.Get()
}

// returnSession returns a session to the pool after resetting it.
func (s *Server) returnSession(session any) {
	s.sessions.Put(session)
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.isListening = false
	s.mu.Unlock()

	err := s.srv.Shutdown(ctx)
	if err != nil {
		return fault.Wrap(err)
	}
	return nil
}

// WrapHandler converts a HandleFunc to an http.Handler using the server's session pool.
func (s *Server) WrapHandler(handler HandleFunc, middlewares []Middleware) http.Handler {
	// Apply middleware
	var handle HandleFunc = handler

	// Reverse the middlewares to run in the desired order
	for i := len(middlewares) - 1; i >= 0; i-- {
		handle = middlewares[i](handle)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.getSession().(*Session)
		if !ok {
			panic("Unable to cast session")
		}
		defer func() {
			sess.reset()
			s.returnSession(sess)
		}()

		sess.init(w, r)

		err := handle(r.Context(), sess)
		if err != nil {
			// Error should have been handled by error middleware
			// If we get here, something went wrong
			panic(err)
		}
	})
}

// SetHandler sets the main HTTP handler for the server.
// This replaces the server's handler with the provided one.
func (s *Server) SetHandler(handler http.Handler) {
	s.handler = handler
	s.srv.Handler = handler
}

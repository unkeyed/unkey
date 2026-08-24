package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	readTimeout     = 10 * time.Second
	writeTimeout    = 20 * time.Second
	shutdownTimeout = 30 * time.Second
)

// Server exposes Prometheus metrics over HTTP.
type Server struct {
	http *http.Server
}

// New creates a server that exposes the default Prometheus registry.
func New() (*Server, error) {
	return newServer(promhttp.Handler()), nil
}

// NewWithRegistry creates a server that exposes metrics from a custom
// prometheus.Registry at the /metrics endpoint.
func NewWithRegistry(reg *prometheus.Registry) (*Server, error) {
	if reg == nil {
		return nil, fmt.Errorf("prometheus: nil registry")
	}

	return newServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})), nil
}

func newServer(metricsHandler http.Handler) *Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metricsHandler)

	return &Server{
		http: &http.Server{
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

// Serve exposes metrics on ln until ctx is canceled or the server stops.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	serveDone := make(chan struct{})
	shutdownErr := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			shutdownErr <- s.Shutdown(shutdownCtx)
		case <-serveDone:
			shutdownErr <- nil
		}
	}()

	err := s.http.Serve(ln)
	close(serveDone)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	return errors.Join(err, <-shutdownErr)
}

// Shutdown gracefully stops the metrics server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// Serve starts a simple HTTP server that exposes Prometheus metrics at GET /metrics.
// The server listens on the provided address (e.g., ":9090" or "127.0.0.1:9090").
//
// This is a package-level alternative to New(). It blocks until the server
// stops or an error occurs.
//
// Example usage:
//
//	go func() {
//	    if err := prometheus.Serve(":9090"); err != nil {
//	        log.Fatalf("Metrics server failed: %v", err)
//	    }
//	}()
func Serve(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	return http.ListenAndServe(addr, mux)
}

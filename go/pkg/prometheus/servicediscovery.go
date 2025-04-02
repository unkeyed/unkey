package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// https://prometheus.io/docs/prometheus/latest/http_sd
// [
//
//	{
//		"targets": [ "<host>", ... ],
//	  "labels": {
//		  "<labelname>": "<labelvalue>", ...
//		}
//	},
//	...
//
// ]
type ServiceDiscoveryResponseElement struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type ServiceDiscoveryResponse = []ServiceDiscoveryResponseElement

type Server struct {
	mu sync.Mutex

	logger      logging.Logger
	isListening bool
	srv         *http.Server
	mux         *http.ServeMux

	sd discovery.Discoverer
}

type Config struct {
	Logger    logging.Logger
	Discovery discovery.Discoverer
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
		srv:         srv,
		mux:         mux,
		sd:          config.Discovery,
	}

	mux.Handle("GET /metrics", promhttp.Handler())

	// dummy metric for the demo
	sdCounterDummyMetric := promauto.NewCounter(prometheus.CounterOpts{
		Name: "sd_called_total",
		Help: "Demo counter just so we have something",
	})

	mux.HandleFunc("GET /sd", func(w http.ResponseWriter, r *http.Request) {
		sdCounterDummyMetric.Add(1.0)
		_, port, err := net.SplitHostPort(s.srv.Addr)
		if err != nil {
			s.internalServerError(err, w)
			return
		}

		addrs, err := s.sd.Discover()
		if err != nil {
			s.internalServerError(err, w)
			return
		}

		e := ServiceDiscoveryResponseElement{
			Targets: []string{},
			Labels: map[string]string{
				// I don't know why they're prefixed but that's what the docs do
				"__meta_region":   "todo",
				"__meta_platform": "aws",
			},
		}

		for _, addr := range addrs {

			e.Targets = append(e.Targets, fmt.Sprintf("%s:%s", addr, port))
		}

		w.Header().Add("Content-Type", "application/json")
		b, err := json.Marshal(ServiceDiscoveryResponse{e})
		if err != nil {
			s.internalServerError(err, w)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			s.logger.Error("unable to write prometheus /sd response",
				"err", err.Error(),
			)
		}
	})

	return s, nil
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

	s.logger.Info("listening", "srv", "prometheus", "addr", addr)

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

func (s *Server) internalServerError(err error, w http.ResponseWriter) {
	s.logger.Error(err.Error())
	w.WriteHeader(http.StatusInternalServerError)
	_, wErr := w.Write([]byte(err.Error()))
	if wErr != nil {
		s.logger.Error("writing response failed", "err", wErr.Error())
	}
}

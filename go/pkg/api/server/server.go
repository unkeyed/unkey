package server

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/api/validation"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type Server struct {
	sync.Mutex
	logger      logging.Logger
	isListening bool
	mux         *http.ServeMux
	srv         *http.Server

	events    EventBuffer
	validator validation.OpenAPIValidator
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
		logger:      config.Logger,
		isListening: false,
		mux:         mux,
		srv:         srv,
	}

	return s, nil
}

// Calling this function multiple times will have no effect.
func (s *Server) Listen(ctx context.Context, addr string) error {
	s.Lock()
	if s.isListening {
		s.logger.Warn(ctx, "already listening")
		s.Unlock()
		return nil
	}
	s.isListening = true
	s.Unlock()
	s.RegisterRoutes()

	s.srv.Addr = addr

	s.logger.Info(ctx, "listening", slog.String("addr", addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() error {
	s.Lock()
	defer s.Unlock()
	return s.srv.Close()

}

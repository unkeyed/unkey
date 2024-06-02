package connect

import (
	"context"
	"fmt"

	"net/http"
	"sync"

	"github.com/bufbuild/connect-go"
	ratelimit "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	ratelimitconnect "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	// "github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/service"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	sync.RWMutex
	isListening    bool
	isShuttingDown bool
	logger         logging.Logger
	shutdownC      chan struct{}
	svc            *service.Service

	ratelimitconnect.UnimplementedRatelimitServiceHandler // returns errors from all methods
}

type Config struct {
	Logger logging.Logger

	Service *service.Service
}

func New(cfg Config) (*Server, error) {

	return &Server{
		logger:         cfg.Logger,
		isListening:    false,
		isShuttingDown: false,
		svc:            cfg.Service,

		shutdownC: make(chan struct{}),
	}, nil
}

func (s *Server) Liveness(ctx context.Context, req *connect.Request[ratelimit.LivenessRequest]) (*connect.Response[ratelimit.LivenessResponse], error) {
	return connect.NewResponse(&ratelimit.LivenessResponse{
		Status: "serving",
	}), nil
}

func (s *Server) Ratelimit(
	ctx context.Context,
	req *connect.Request[ratelimit.RatelimitRequest],
) (*connect.Response[ratelimit.RatelimitResponse], error) {
	// err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	// if err != nil {
	// 	return nil, err
	// }

	res, err := s.svc.Ratelimit(ctx, req.Msg)
	if err != nil {
		s.logger.Err(err).Msg("failed to ratelimit")
		return nil, fmt.Errorf("failed to ratelimit: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *Server) Listen(addr string) error {
	s.Lock()
	if s.isListening {
		s.logger.Info().Msg("already listening")
		s.Unlock()
		return nil
	}
	s.isListening = true
	s.Unlock()

	mux := http.NewServeMux() // `NewServeMux` is a function in the `net/http` package in Go that creates a new HTTP request multiplexer (ServeMux). A ServeMux is an HTTP request router that matches the URL of incoming requests against a list of registered patterns and calls the handler for the pattern that most closely matches the URL path. It essentially acts as a router for incoming HTTP requests, directing them to the appropriate handler based on the URL path.NewServeMux()

	mux.Handle(ratelimitconnect.NewRatelimitServiceHandler(s))
	mux.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to write response")
		}
	})

	srv := &http.Server{Addr: addr, Handler: h2c.NewHandler(mux, &http2.Server{})}

	s.logger.Info().Str("addr", addr).Msg("listening")
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			s.logger.Error().Err(err).Msg("listen and serve failed")
		}
	}()

	<-s.shutdownC
	s.logger.Info().Msg("shutting down")
	return srv.Shutdown(context.Background())

}

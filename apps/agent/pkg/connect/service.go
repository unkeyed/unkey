package connect

import (
	"context"
	"encoding/json"
	"sync"

	"net/http"

	"github.com/bufbuild/connect-go"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Service interface {
	CreateHandler() (pattern string, handler http.Handler)
}

type Server struct {
	sync.Mutex
	logger         logging.Logger
	mux            *http.ServeMux
	shutdownC      chan struct{}
	isShuttingDown bool
	isListening    bool
	image          string
}

type Config struct {
	Logger logging.Logger
	Image  string
}

func New(cfg Config) (*Server, error) {

	return &Server{
		logger:         cfg.Logger,
		isListening:    false,
		isShuttingDown: false,
		mux:            http.NewServeMux(),
		image:          cfg.Image,

		shutdownC: make(chan struct{}),
	}, nil
}

func (s *Server) AddService(svc Service) {
	pattern, handler := svc.CreateHandler()
	s.logger.Info().Str("pattern", pattern).Msg("adding service")

	h := newTracingMiddleware(newHeaderMiddleware(newLoggingMiddleware(handler, s.logger)))
	s.mux.Handle(pattern, h)
}

func (s *Server) Liveness(ctx context.Context, req *connect.Request[ratelimitv1.LivenessRequest]) (*connect.Response[ratelimitv1.LivenessResponse], error) {
	return connect.NewResponse(&ratelimitv1.LivenessResponse{
		Status: "serving",
	}), nil
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

	s.mux.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {

		b, err := json.Marshal(map[string]string{"status": "serving", "image": s.image})
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to marshal response")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(b)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to write response")
		}
	})

	srv := &http.Server{Addr: addr, Handler: h2c.NewHandler(s.mux, &http2.Server{})}

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

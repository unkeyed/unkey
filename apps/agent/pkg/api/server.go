package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/services/eventrouter"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
)

type Server struct {
	sync.Mutex
	logger      logging.Logger
	metrics     metrics.Metrics
	isListening bool
	api         huma.API
	mux         *http.ServeMux
	srv         *http.Server

	// The bearer token required for inter service communication
	authToken string
	Vault     *vault.Service
	Ratelimit ratelimit.Service
}

type Config struct {
	Logger  logging.Logger
	Metrics metrics.Metrics
}

func New(config Config) *Server {
	mux := http.NewServeMux()
	s := &Server{
		logger:      config.Logger,
		metrics:     config.Metrics,
		isListening: false,
		api:         humago.New(mux, huma.DefaultConfig("Unkey API", "1.0.0")),
		mux:         mux,
	}

	s.api.UseMiddleware(func(hCtx huma.Context, next func(huma.Context)) {

		start := time.Now()

		s.logger.Info().Str("method", hCtx.Method()).Str("path", hCtx.URL().Path).Msg("request")
		ctx, span := tracing.Start(hCtx.Context(), "api.request")
		defer span.End()

		next(huma.WithContext(hCtx, ctx))

		s.metrics.Record(metrics.HttpRequest{
			Method:         hCtx.Method(),
			Path:           hCtx.URL().Path,
			ServiceLatency: time.Since(start).Milliseconds(),
			UserAgent:      hCtx.Header("user-agent"),
			RemoteAddr:     hCtx.RemoteAddr(),
			SourceIP:       hCtx.Header("Fly-Client-IP"),
		})
	})

	return s
}

func (s *Server) HumaAPI() huma.API {
	return s.api
}

func (s *Server) Services() routes.Services {
	return routes.Services{
		Logger:    s.logger,
		Metrics:   s.metrics,
		Vault:     s.Vault,
		Ratelimit: s.Ratelimit,
	}
}

func (s *Server) WithEventRouter(svc *eventrouter.Service) {
	s.Lock()
	defer s.Unlock()

	route, handler := svc.CreateHandler()

	s.mux.Handle(route, handler)

}

// Calling this function multiple times will have no effect.
func (s *Server) Listen(addr string) error {
	s.Lock()
	if s.isListening {
		s.logger.Warn().Msg("already listening")
		s.Unlock()
		return nil
	}
	s.isListening = true
	s.Unlock()
	s.srv = &http.Server{Addr: addr, Handler: s.mux}

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
	s.srv.ReadTimeout = 10 * time.Second
	s.srv.WriteTimeout = 20 * time.Second

	s.logger.Info().Str("addr", addr).Msg("listening")
	return s.srv.ListenAndServe()

}

func (s *Server) Shutdown() error {
	s.Lock()
	defer s.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)

}

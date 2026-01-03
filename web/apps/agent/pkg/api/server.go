package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/api/validation"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
)

type Server struct {
	sync.Mutex
	logger      logging.Logger
	metrics     metrics.Metrics
	isListening bool
	mux         *http.ServeMux
	srv         *http.Server

	// The bearer token required for inter service communication
	authToken string
	vault     *vault.Service
	ratelimit ratelimit.Service

	clickhouse EventBuffer
	validator  validation.OpenAPIValidator
}

type Config struct {
	NodeId     string
	Logger     logging.Logger
	Metrics    metrics.Metrics
	Ratelimit  ratelimit.Service
	Clickhouse EventBuffer
	Vault      *vault.Service
	AuthToken  string
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
		metrics:     config.Metrics,
		ratelimit:   config.Ratelimit,
		vault:       config.Vault,
		isListening: false,
		mux:         mux,
		srv:         srv,
		clickhouse:  config.Clickhouse,
		authToken:   config.AuthToken,
	}
	// validationMiddleware, err := s.createOpenApiValidationMiddleware("./pkg/openapi/openapi.json")
	// if err != nil {
	// 	return nil, fault.Wrap(err, fmsg.With("openapi spec encountered an error"))
	// }
	// s.app.Use(
	// 	createLoggerMiddleware(s.logger),
	// 	createMetricsMiddleware(),
	// 	// validationMiddleware,
	// )
	// s.app.Use(tracingMiddleware)
	v, err := validation.New()
	if err != nil {
		return nil, err
	}
	s.validator = v

	s.srv.Handler = withMetrics(withTracing(withRequestId(s.mux)))

	return s, nil
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
	s.RegisterRoutes()

	s.srv.Addr = addr

	s.logger.Info().Str("addr", addr).Msg("listening")
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() error {
	s.Lock()
	defer s.Unlock()
	return s.srv.Close()

}

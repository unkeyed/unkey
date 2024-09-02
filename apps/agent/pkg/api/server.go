package api

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/ftag"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/validation"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/services/eventrouter"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
)

type Server struct {
	sync.Mutex
	logger      logging.Logger
	metrics     metrics.Metrics
	isListening bool

	app *fiber.App

	// The bearer token required for inter service communication
	authToken string
	Vault     *vault.Service
	Ratelimit ratelimit.Service

	clickhouse EventBuffer
	validator  validation.OpenAPIValidator
}

type Config struct {
	NodeId     string
	Logger     logging.Logger
	Metrics    metrics.Metrics
	Clickhouse EventBuffer
}

func New(config Config) (*Server, error) {

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
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

		ErrorHandler: func(c *fiber.Ctx, err error) error {

			code := fiber.StatusInternalServerError
			tag := ftag.Get(err)
			switch tag {
			case ftag.NotFound:
				code = fiber.StatusNotFound

			}

			issues := fault.Flatten(err)

			errors := make([]openapi.ErrorDetail, len(issues))
			for i, issue := range issues {
				errors[i] = openapi.ErrorDetail{
					Location: issue.Location,
					Message:  issue.Message,
				}
			}

			body := openapi.ErrorModel{
				Detail:    "",
				Errors:    errors,
				Instance:  string(tag),
				RequestId: "TODO", //c.UserContext().Value("requestID").(string),
				Status:    code,
				Title:     "",
				Type:      "https://tools.ietf.org/html/rfc7231#section-6.6.1",
			}
			b, err := json.MarshalIndent(body, "", "  ")
			if err == nil {
				config.Logger.Info().Str("error", string(b)).Msg("returning error")
			}

			return c.Status(code).JSON(body)
		},
	})

	// app.Use(recover.New())

	s := &Server{
		logger:      config.Logger,
		metrics:     config.Metrics,
		isListening: false,
		app:         app,
		clickhouse:  config.Clickhouse,
	}
	// validationMiddleware, err := s.createOpenApiValidationMiddleware("./pkg/openapi/openapi.json")
	// if err != nil {
	// 	return nil, fault.Wrap(err, fmsg.With("openapi spec encountered an error"))
	// }
	s.app.Use(
		createLoggerMiddleware(s.logger),
		createMetricsMiddleware(),
		// validationMiddleware,
	)
	// s.app.Use(tracingMiddleware)
	v, err := validation.New("./pkg/openapi/openapi.json")
	if err != nil {
		return nil, err
	}
	s.validator = v

	s.RegisterRoutes()
	return s, nil
}

func (s *Server) WithEventRouter(svc *eventrouter.Service) {
	s.Lock()
	defer s.Unlock()

	route, handler := svc.CreateHandler()

	s.app.Use(route, handler)

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

	s.logger.Info().Str("addr", addr).Msg("listening")
	return s.app.Listen(addr)

}

func (s *Server) Shutdown() error {
	s.Lock()
	defer s.Unlock()
	return s.app.Shutdown()

}

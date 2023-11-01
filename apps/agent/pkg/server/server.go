package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/gofiber/fiber/v2"

	"github.com/go-playground/validator/v10"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"

	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Logger   logging.Logger
	KeyCache cache.Cache[*authenticationv1.Key]
	// The ApiCache uses the KeyAuthId as cache key, not an apiId
	ApiCache          cache.Cache[*apisv1.Api]
	Database          database.Database
	Ratelimit         ratelimit.Ratelimiter
	Tracer            trace.Tracer
	Analytics         analytics.Analytics
	UnkeyAppAuthToken string
	UnkeyWorkspaceId  string
	UnkeyApiId        string
	UnkeyKeyAuthId    string
	Region            string
	EventBus          events.EventBus
	Version           string
	WorkspaceService  workspaces.WorkspaceService
	ApiService        apis.ApiService
	KeyService        keys.KeyService
	Metrics           metrics.Metrics
}

type Server struct {
	App       *fiber.App
	logger    logging.Logger
	validator *validator.Validate
	db        database.Database
	keyCache  cache.Cache[*authenticationv1.Key]
	apiCache  cache.Cache[*apisv1.Api]
	ratelimit ratelimit.Ratelimiter

	// Not guaranteed to be available, always do a nil check first!
	tracer trace.Tracer
	// Not guaranteed to be available, always do a nil check first!
	analytics         analytics.Analytics
	unkeyAppAuthToken string
	unkeyWorkspaceId  string
	unkeyApiId        string
	unkeyKeyAuthId    string
	region            string
	metrics           metrics.Metrics

	// Used for communication with other pods
	// Not guaranteed to be available, always do a nil check first!
	events  events.EventBus
	version string

	workspaceService workspaces.WorkspaceService
	apiService       apis.ApiService
	keyService       keys.KeyService
}

// New creates a new server
func New(config Config) *Server {

	appConfig := fiber.Config{
		DisableStartupMessage: true,
		Immutable:             true,
	}

	s := &Server{
		App:               fiber.New(appConfig),
		events:            config.EventBus,
		logger:            config.Logger,
		validator:         validator.New(),
		db:                config.Database,
		keyCache:          config.KeyCache,
		apiCache:          config.ApiCache,
		ratelimit:         config.Ratelimit,
		tracer:            config.Tracer,
		analytics:         config.Analytics,
		unkeyAppAuthToken: config.UnkeyAppAuthToken,
		unkeyWorkspaceId:  config.UnkeyWorkspaceId,
		unkeyApiId:        config.UnkeyApiId,
		unkeyKeyAuthId:    config.UnkeyKeyAuthId,
		region:            config.Region,
		version:           config.Version,
		workspaceService:  config.WorkspaceService,
		apiService:        config.ApiService,
		keyService:        config.KeyService,

		metrics: config.Metrics,
	}
	if s.events == nil {
		s.events = events.NewNoop()
	}
	if s.analytics == nil {
		s.analytics = analytics.NewNoop()
	}

	s.App.Use(recover.New(recover.Config{EnableStackTrace: true, StackTraceHandler: func(c *fiber.Ctx, err interface{}) {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, false)]
		config.Logger.Error().Any("err", err).Msg(string(buf))
	}}))

	s.App.Use(cors.New())

	basicAuthUser := os.Getenv("BASIC_AUTH_USER")
	basicAuthPassword := os.Getenv("BASIC_AUTH_PASSWORD")
	if basicAuthUser != "" && basicAuthPassword != "" {
		users := map[string]string{}
		users[basicAuthUser] = basicAuthPassword

		s.App.All("/debug/*", basicauth.New(basicauth.Config{
			Users: users,
		}),
			pprof.New(),
		)
	}
	s.App.Use(func(c *fiber.Ctx) error {
		if c.Path() == "/v1/liveness" {
			return c.Next()
		}

		// This header is a three letter region code which represents the region that the connection was accepted in and routed from.
		edgeRegion := c.Get("Fly-Region")

		ctx, span := s.tracer.Start(c.UserContext(), "request", trace.WithAttributes(
			attribute.String("method", c.Route().Method),
			attribute.String("path", c.Path()),
			attribute.String("edgeRegion", edgeRegion),
			attribute.String("region", s.region),
		))
		defer span.End()
		c.SetUserContext(ctx)
		traceId := span.SpanContext().TraceID().String()

		c.Set("Unkey-Trace-Id", fmt.Sprintf("%s:%s::%s", s.region, edgeRegion, traceId))
		c.Set("Unkey-Version", s.version)
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		error := ""
		if err != nil {
			error = err.Error()
		}
		if s.metrics != nil {
			s.metrics.ReportHttpRequest(metrics.HttpRequestReport{
				Method:         c.Route().Method,
				Status:         c.Response().StatusCode(),
				Path:           c.Path(),
				EdgeRegion:     edgeRegion,
				TraceId:        traceId,
				ServiceLatency: latency.Milliseconds(),
				Error:          error,
			})
		}

		log := config.Logger.With().
			Str("body", string(c.Response().Body())).
			Str("method", c.Route().Method).
			Int("status", c.Response().StatusCode()).
			Str("path", c.Path()).
			Int64("serviceLatency", latency.Milliseconds()).
			Str("edgeRegion", edgeRegion).
			Str("traceId", traceId).
			Logger()

		if c.Response().StatusCode() >= 500 ||

			(err != nil &&
				!errors.Is(err, fiber.ErrMethodNotAllowed) &&
				!errors.Is(err, fiber.ErrNotFound)) {
			log.Err(err).Msg("request failed")
			span.RecordError(err)
		} else {
			log.Info().Msg("request completed")
		}
		return err
	})

	s.App.Get("/v1/liveness", s.liveness)

	// Used internally only, not covered by versioning
	s.App.Post("/v1/internal.createRootKey", s.createRootKey)
	s.App.Post("/v1/internal.removeRootKey", s.deleteRootKey)

	// workspaceService
	s.App.Post("/v1/workspaces.createWorkspace", s.v1CreateWorkspace)

	// apiService
	s.App.Post("/v1/apis.createApi", s.v1CreateApi)
	s.App.Post("/v1/apis.deleteApi", s.v1DeleteApi)
	s.App.Get("/v1/apis.findApi", s.getApi)
	s.App.Get("/v1/apis.listKeys", s.listKeys)

	// keyService
	s.App.Post("/v1/keys.createKey", s.v1CreateKey)
	s.App.Post("/v1/keys.verifyKey", s.v1VerifyKey)
	s.App.Post("/v1/keys.removeKey", s.v1RemoveKey)
	s.App.Post("/v1/keys.updateKey", s.updateKey)
	s.App.Get("/v1/keys.findKey", s.v1FindKey)

	// legacy
	s.App.Post("/v1/keys", s.v1CreateKey)
	s.App.Get("/v1/keys/:keyId", s.getKey)
	s.App.Put("/v1/keys/:keyId", s.updateKey)
	s.App.Delete("/v1/keys/:keyId", s.deleteKey)
	s.App.Post("/v1/keys/verify", s.v1VerifyKey)

	s.App.Get("/v1/apis/:apiId", s.getApi)
	s.App.Get("/v1/apis/:apiId/keys", s.listKeys)

	s.App.Post("/v1/internal/rootkeys", s.createRootKey)

	// experimental
	s.App.Get("/vx/keys/:keyId/stats", s.getKeyStats)

	return s
}

func (s *Server) Start(addr string) error {
	s.logger.Info().Str("addr", addr).Msg("listening")
	err := s.App.Listen(addr)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("api server error: %s", err.Error())
	}
	return nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	s.logger.Info().Msg("stopping..")
	defer s.logger.Info().Msg("stopped")
	return s.App.Server().ShutdownWithContext(ctx)
}

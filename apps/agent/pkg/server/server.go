package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/gofiber/fiber/v2"

	"github.com/go-playground/validator/v10"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Logger   logging.Logger
	KeyCache cache.Cache[entities.Key]
	// The ApiCache uses the KeyAuthId as cache key, not an apiId
	ApiCache          cache.Cache[entities.Api]
	Database          database.Database
	Ratelimit         ratelimit.Ratelimiter
	GlobalRatelimit   ratelimit.Ratelimiter
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
}

type Server struct {
	app             *fiber.App
	logger          logging.Logger
	validator       *validator.Validate
	db              database.Database
	keyCache        cache.Cache[entities.Key]
	apiCache        cache.Cache[entities.Api]
	ratelimit       ratelimit.Ratelimiter
	globalRatelimit ratelimit.Ratelimiter

	// Not guaranteed to be available, always do a nil check first!
	tracer trace.Tracer
	// Not guaranteed to be available, always do a nil check first!
	analytics         analytics.Analytics
	unkeyAppAuthToken string
	unkeyWorkspaceId  string
	unkeyApiId        string
	unkeyKeyAuthId    string
	region            string

	// Used for communication with other pods
	// Not guaranteed to be available, always do a nil check first!
	events  events.EventBus
	version string

	workspaceService workspaces.WorkspaceService
}

func New(config Config) *Server {

	appConfig := fiber.Config{
		DisableStartupMessage: true,
		Immutable:             true,
	}

	s := &Server{
		app:               fiber.New(appConfig),
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
	}
	if s.events == nil {
		s.events = events.NewNoop()
	}
	if s.analytics == nil {
		s.analytics = analytics.NewNoop()
	}

	s.app.Use(recover.New(recover.Config{EnableStackTrace: true, StackTraceHandler: func(c *fiber.Ctx, err interface{}) {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, false)]
		config.Logger.Error("recovered from panic", zap.Any("err", err), zap.ByteString("stacktrace", buf))
	}}))

	s.app.Use(func(c *fiber.Ctx) error {
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

		log := config.Logger.With(
			zap.String("method", c.Route().Method),
			zap.Int("status", c.Response().StatusCode()),
			zap.String("path", c.Path()),
			zap.Error(err),
			zap.Int64("serviceLatency", latency.Milliseconds()),
			zap.String("edgeRegion", edgeRegion),
			zap.String("traceId", traceId),
		)

		if err != nil || c.Response().StatusCode() >= 500 {
			log.Error("request failed", zap.String("body", string(c.Response().Body())), zap.Error(err))
			span.RecordError(err)
		} else {
			log.Info("request completed")
		}

		return err
	})

	s.app.Get("/v1/liveness", s.liveness)

	// Used internally only, not covered by versioning
	s.app.Post("/v1/internal/rootkeys", s.createRootKey)

	s.app.Post("/v1/keys", s.createKey)
	s.app.Get("/v1/keys/:keyId", s.getKey)
	s.app.Put("/v1/keys/:keyId", s.updateKey)
	s.app.Delete("/v1/keys/:keyId", s.deleteKey)
	s.app.Post("/v1/keys/verify", s.verifyKey)

	s.app.Get("/v1/apis/:apiId", s.getApi)
	s.app.Get("/v1/apis/:apiId/keys", s.listKeys)

	s.app.Get("/vx/keys/:keyId/stats", s.getKeyStats)

	// new RPC style endpoints
	s.app.Post("/v1/workspaces.create", s.createWorkspace)

	return s
}

func (s *Server) Start(addr string) error {
	s.logger.Info("listening", zap.String("addr", addr))
	err := s.app.Listen(addr)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("api server error: %s", err.Error())
	}
	return nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	s.logger.Info("stopping..")
	defer s.logger.Info("stopped")
	return s.app.Server().ShutdownWithContext(ctx)
}

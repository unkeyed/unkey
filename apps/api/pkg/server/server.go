package server

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/gofiber/fiber/v2"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/ratelimit"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Logger    logging.Logger
	Cache     cache.Cache[entities.Key]
	Database  database.Database
	Ratelimit *ratelimit.Ratelimiter
	Tracer    trace.Tracer
}

type Server struct {
	app       *fiber.App
	logger    logging.Logger
	validator *validator.Validate
	db        database.Database
	cache     cache.Cache[entities.Key]
	ratelimit *ratelimit.Ratelimiter
	tracer    trace.Tracer
}

func New(config Config) *Server {
	appConfig := fiber.Config{
		DisableStartupMessage: true,
		Immutable:             true,
	}

	s := &Server{
		app:       fiber.New(appConfig),
		logger:    config.Logger,
		validator: validator.New(),
		db:        config.Database,
		cache:     config.Cache,
		ratelimit: config.Ratelimit,
		tracer:    config.Tracer,
	}

	s.app.Use(recover.New(recover.Config{EnableStackTrace: true, StackTraceHandler: func(c *fiber.Ctx, err interface{}) {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, false)]
		config.Logger.Error("recovered from panic", zap.Any("err", err), zap.ByteString("stacktrace", buf))
	}}))

	s.app.Use(func(c *fiber.Ctx) error {
		// This header is a three letter region code which represents the region that the connection was accepted in and routed from.
		edgeRegion := c.Get("Fly-Region")

		ctx, span := s.tracer.Start(c.UserContext(), "request", trace.WithAttributes(
			attribute.String("method", c.Route().Method),
			attribute.String("path", c.Path()),
			attribute.String("edgeRegion", edgeRegion),
		))
		defer span.End()
		c.SetUserContext(ctx)
		c.Set("Unkey-Trace-Id", span.SpanContext().TraceID().String())
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		log := config.Logger.With(
			zap.String("method", c.Route().Method),
			zap.Int("status", c.Response().StatusCode()),
			zap.String("path", c.Path()),
			zap.Error(err),
			zap.Duration("ms", latency),
			zap.String("edgeRegion", edgeRegion),
		)

		if err != nil {
			log.Error("request failed")
			span.RecordError(err)
		} else {
			log.Info("request completed")
		}

		return err
	})

	s.app.Get("/v1/liveness", s.liveness)

	s.app.Post("/v1/keys", s.createKey)
	s.app.Delete("/v1/keys/:keyId", s.deleteKey)
	s.app.Post("/v1/keys/verify", s.verifyKey)

	s.app.Get("/v1/apis/:apiId", s.getApi)
	s.app.Get("/v1/apis/:apiId/keys", s.listKeys)

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
	return s.app.Server().Shutdown()
}

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
	"github.com/chronark/unkey/apps/api/pkg/kafka"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/ratelimit"
	"github.com/chronark/unkey/apps/api/pkg/tinybird"
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
	// Potentially the user does not have tinybird set up or does not want to use it
	// simply pass in nil in that case
	Tinybird          *tinybird.Tinybird
	UnkeyAppAuthToken string
	UnkeyWorkspaceId  string
	UnkeyApiId        string
	Region            string
	Kafka             *kafka.Kafka
}

type Server struct {
	app       *fiber.App
	logger    logging.Logger
	validator *validator.Validate
	db        database.Database
	cache     cache.Cache[entities.Key]
	ratelimit *ratelimit.Ratelimiter
	tracer    trace.Tracer
	// potentially nil, always do a check first
	tinybird       *tinybird.Tinybird
	verificationsC chan tinybird.KeyVerificationEvent
	closeC         chan struct{}
	// Used to authenticate our frontend when creating new unkey keys.
	unkeyAppAuthToken string
	unkeyWorkspaceId  string
	unkeyApiId        string
	region            string
	kafka             *kafka.Kafka
}

func New(config Config) *Server {
	appConfig := fiber.Config{
		DisableStartupMessage: true,
		Immutable:             true,
	}

	s := &Server{
		app:               fiber.New(appConfig),
		kafka:             config.Kafka,
		logger:            config.Logger,
		validator:         validator.New(),
		db:                config.Database,
		cache:             config.Cache,
		ratelimit:         config.Ratelimit,
		tracer:            config.Tracer,
		tinybird:          config.Tinybird,
		verificationsC:    make(chan tinybird.KeyVerificationEvent),
		closeC:            make(chan struct{}),
		unkeyAppAuthToken: config.UnkeyAppAuthToken,
		unkeyWorkspaceId:  config.UnkeyWorkspaceId,
		unkeyApiId:        config.UnkeyApiId,
		region:            config.Region,
	}

	if config.Tinybird != nil {
		go s.SyncTinybird()
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

		c.Set("Unkey-Trace-Id", fmt.Sprintf("%s:%s::%s", s.region, edgeRegion, span.SpanContext().TraceID().String()))
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

	// Used internally only, not covered by versioning
	s.app.Post("/v1/internal/rootkeys", s.createRootKey)

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
	s.closeC <- struct{}{}
	return s.app.Server().Shutdown()
}

// Call this in a goroutine
func (s *Server) SyncTinybird() {
	for {
		select {
		case <-s.closeC:
			return
		case e := <-s.verificationsC:
			err := s.tinybird.PublishKeyVerificationEvent("key_verifications__v1", e)
			if err != nil {
				s.logger.Error("unable to publish event to tinybird",
					zap.String("workspaceId", e.WorkspaceId),
					zap.String("apiId", e.ApiId),
					zap.String("keyId", e.KeyId),
				)
			}
			return
		}
	}
}

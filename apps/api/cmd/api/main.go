package main

import (
	"context"
	"fmt"
	"github.com/chronark/unkey/apps/api/pkg/cache"
	cacheMiddleware "github.com/chronark/unkey/apps/api/pkg/cache/middleware"
	"github.com/chronark/unkey/apps/api/pkg/database"
	databaseMiddleware "github.com/chronark/unkey/apps/api/pkg/database/middleware"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/env"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/ratelimit"
	"github.com/chronark/unkey/apps/api/pkg/server"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

// Set when we build the docker image
var (
	version string
)

func main() {

	debug := os.Getenv("DEBUG") != ""
	logger := logging.New()
	if !debug {
		logger = logger.With(zap.String("version", version))
	}
	defer func() {
		_ = logger.Sync()
	}()
	logger.Info("Starting Unkey API Server")

	e := env.Env{
		ErrorHandler: func(err error) { logger.Fatal("unable to load environment variable", zap.Error(err)) },
	}
	region := e.String("FLY_REGION", "local")
	logger = logger.With(zap.String("region", region))

	axiomOrgId := e.String("AXIOM_ORG_ID", "")
	axiomToken := e.String("AXIOM_TOKEN", "")

	var tracer tracing.Tracer
	if axiomOrgId != "" && axiomToken != "" {

		t, closeTracer, err := tracing.New(context.Background(), tracing.Config{
			Dataset:    "tracing",
			Service:    "api",
			Version:    version,
			AxiomOrgId: e.String("AXIOM_ORG_ID"),
			AxiomToken: e.String("AXIOM_TOKEN"),
		})
		if err != nil {
			logger.Fatal("unable to start tracer", zap.Error(err))
		}
		defer func() {
			err := closeTracer()
			if err != nil {
				logger.Fatal("unable to close tracer", zap.Error(err))
			}
		}()
		tracer = t
		logger.Info("Axiom tracing enabled")
	} else {
		tracer = tracing.NewNoop()
	}

	r := ratelimit.New()

	db, err := database.New(database.Config{
		PrimaryUs:   e.String("DATABASE_DSN"),
		ReplicaEu:   e.String("DATABASE_DSN_EU", ""),
		ReplicaAsia: e.String("DATABASE_DSN_ASIA", ""),
		FlyRegion:   region,
	})
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	db = databaseMiddleware.WithTracing(db, tracer)

	port := e.String("PORT", "8080")

	c := cache.NewInMemoryCache[entities.Key]()
	c = cacheMiddleware.WithTracing[entities.Key](c, tracer)

	srv := server.New(server.Config{
		Logger:    logger,
		Cache:     c,
		Database:  db,
		Ratelimit: r,
		Tracer:    tracer,
	})

	err = srv.Start(fmt.Sprintf("0.0.0.0:%s", port))
	defer func() {
		closeErr := srv.Close()
		if closeErr != nil {
			logger.Fatal("Failed to close server", zap.Error(closeErr))
		}
	}()
	if err != nil {
		logger.Fatal("Failed to run service", zap.Error(err))
	}

	cShutdown := make(chan os.Signal, 1)
	signal.Notify(cShutdown, os.Interrupt, syscall.SIGTERM)

	// wait for signal
	sig := <-cShutdown
	logger.Warn("Caught signal", zap.Any("sig", sig))
}

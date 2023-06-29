package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/env"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/ratelimit"
	"github.com/chronark/unkey/apps/api/pkg/server"
	"go.uber.org/zap"
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
	defer logger.Sync()
	logger.Info("Starting Unkey API Server")

	e := env.Env{
		ErrorHandler: func(err error) { logger.Fatal("unable to load environment variable", zap.Error(err)) },
	}
	region := e.String("FLY_REGION")
	logger = logger.With(zap.String("region", region))

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

	port := e.String("PORT", "8080")

	srv := server.New(server.Config{
		Logger:    logger,
		Cache:     cache.NewInMemoryCache[entities.Key](time.Minute),
		Database:  db,
		Ratelimit: r,
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

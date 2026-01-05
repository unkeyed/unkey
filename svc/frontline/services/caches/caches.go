package caches

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Caches holds all cache instances used throughout frontline.
type Caches struct {
	// HostName -> frontline Route
	FrontlineRoutes cache.Cache[string, db.FrontlineRoute]

	// EnvironmentID -> List of Sentinels
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]

	// HostName -> Certificate
	TLSCertificates cache.Cache[string, tls.Certificate]
}

// Config defines the configuration options for initializing caches.
type Config struct {
	Logger logging.Logger
	Clock  clock.Clock
}

func New(config Config) (Caches, error) {
	frontlineRoute, err := cache.New(cache.Config[string, db.FrontlineRoute]{
		Fresh:    time.Second * 5,
		Stale:    time.Second * 30,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "frontline_route",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create sentinel config cache: %w", err)
	}

	sentinelsByEnvironment, err := cache.New(cache.Config[string, []db.Sentinel]{
		Fresh:    time.Second * 10,
		Stale:    time.Minute,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "sentinels_by_environment",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create instances by deployment cache: %w", err)
	}

	tlsCertificate, err := cache.New(cache.Config[string, tls.Certificate]{
		Fresh:    time.Hour,
		Stale:    time.Hour * 12,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "tls_certificate",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create certificate cache: %w", err)
	}

	return Caches{
		FrontlineRoutes:        middleware.WithTracing(frontlineRoute),
		SentinelsByEnvironment: middleware.WithTracing(sentinelsByEnvironment),
		TLSCertificates:        middleware.WithTracing(tlsCertificate),
	}, nil
}

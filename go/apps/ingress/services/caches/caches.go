package caches

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all cache instances used throughout ingress.
type Caches struct {
	// HostName -> IngressRoute
	IngressRoutes cache.Cache[string, db.IngressRoute]

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
	ingressRoute, err := cache.New(cache.Config[string, db.IngressRoute]{
		Fresh:    time.Second * 5,
		Stale:    time.Second * 30,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "ingress_route",
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
		IngressRoutes:          middleware.WithTracing(ingressRoute),
		SentinelsByEnvironment: middleware.WithTracing(sentinelsByEnvironment),
		TLSCertificates:        middleware.WithTracing(tlsCertificate),
	}, nil
}

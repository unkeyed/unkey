package caches

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all cache instances used throughout ingress.
type Caches struct {
	// HostName -> Certificate
	TLSCertificate cache.Cache[string, tls.Certificate]
}

// Config defines the configuration options for initializing caches.
type Config struct {
	// Logger is used for logging cache operations and errors.
	Logger logging.Logger

	// Clock provides time functionality, allowing easier testing.
	Clock clock.Clock
}

// New creates and initializes all cache instances with appropriate settings.
func New(config Config) (Caches, error) {
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
		TLSCertificate: middleware.WithTracing(tlsCertificate),
	}, nil
}

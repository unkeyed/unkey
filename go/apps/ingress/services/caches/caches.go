package caches

import (
	"crypto/tls"
	"fmt"
	"time"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/cache/middleware"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// GatewayConfigData holds gateway configuration with workspace ID
type GatewayConfigData struct {
	Config      *partitionv1.GatewayConfig
	WorkspaceID string
}

// Caches holds all cache instances used throughout ingress.
type Caches struct {
	// HostName -> Gateway Configuration
	GatewayConfig cache.Cache[string, GatewayConfigData]

	// DeploymentID -> List of Instances
	InstancesByDeployment cache.Cache[string, []db.Vm]

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
	gatewayConfig, err := cache.New(cache.Config[string, GatewayConfigData]{
		Fresh:    time.Second * 5,
		Stale:    time.Second * 30,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "gateway_config",
		Clock:    config.Clock,
	})
	if err != nil {
		return Caches{}, fmt.Errorf("failed to create gateway config cache: %w", err)
	}

	instancesByDeployment, err := cache.New(cache.Config[string, []db.Vm]{
		Fresh:    time.Second * 10,
		Stale:    time.Minute,
		Logger:   config.Logger,
		MaxSize:  10_000,
		Resource: "instances_by_deployment",
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
		GatewayConfig:         middleware.WithTracing(gatewayConfig),
		InstancesByDeployment: middleware.WithTracing(instancesByDeployment),
		TLSCertificate:        middleware.WithTracing(tlsCertificate),
	}, nil
}

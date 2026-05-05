package caches

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/bus"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// Caches holds all cache instances used throughout frontline.
type Caches struct {
	// HostName -> frontline Route
	FrontlineRoutes cache.Cache[string, db.FindFrontlineRouteByFQDNRow]

	// DeploymentID -> List of Instances
	InstancesByDeployment cache.Cache[string, []db.FindInstancesByDeploymentIDRow]

	// DeploymentID -> Parsed sentinel policies. Cached to avoid re-parsing
	// the protojson SentinelConfig on every request.
	Policies cache.Cache[string, []*frontlinev1.Policy]

	// HostName -> Certificate
	TLSCertificates cache.Cache[string, tls.Certificate]
}

// Close is a no-op kept for API stability. Cache subscriptions are torn
// down by closing the bus, which is owned by the caller.
func (c *Caches) Close() error { return nil }

// Config defines the configuration options for initializing caches.
type Config struct {
	Clock clock.Clock

	// Bus is the event bus used for distributed cache invalidation. Pass
	// bus.NewNoop() in processes that have no gossip configured.
	Bus bus.Bus

	// NodeID identifies this node in the cluster.
	NodeID string
}

type clusterOpts[K comparable] struct {
	bus         bus.Bus
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
}

func createCache[K comparable, V any](
	cacheConfig cache.Config[K, V],
	opts clusterOpts[K],
) (cache.Cache[K, V], error) {
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Bus:         opts.bus,
		NodeID:      opts.nodeID,
		KeyToString: opts.keyToString,
		StringToKey: opts.stringToKey,
	})
	if err != nil {
		return nil, err
	}

	return clusterCache, nil
}

func New(config Config) (*Caches, error) {
	if config.Bus == nil {
		config.Bus = bus.NewNoop()
	}

	if config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		config.NodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	stringKeyOpts := clusterOpts[string]{
		bus:         config.Bus,
		nodeID:      config.NodeID,
		keyToString: nil,
		stringToKey: nil,
	}

	frontlineRoute, err := createCache(
		cache.Config[string, db.FindFrontlineRouteByFQDNRow]{
			Fresh:    5 * time.Second,
			Stale:    5 * time.Minute,
			MaxSize:  10_000,
			Resource: "frontline_route",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create frontline route cache: %w", err)
	}

	policies, err := createCache(
		cache.Config[string, []*frontlinev1.Policy]{
			Fresh:    30 * time.Second,
			Stale:    5 * time.Minute,
			MaxSize:  10_000,
			Resource: "policies",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create policies cache: %w", err)
	}

	instancesByDeployment, err := createCache(
		cache.Config[string, []db.FindInstancesByDeploymentIDRow]{
			Fresh:    10 * time.Second,
			Stale:    60 * time.Second,
			MaxSize:  10_000,
			Resource: "instances_by_deployment",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create instances by deployment cache: %w", err)
	}

	tlsCertificate, err := createCache(
		cache.Config[string, tls.Certificate]{
			Fresh:    time.Hour,
			Stale:    time.Hour * 12,
			MaxSize:  10_000,
			Resource: "tls_certificate",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate cache: %w", err)
	}

	return &Caches{
		FrontlineRoutes:       middleware.WithTracing(frontlineRoute),
		InstancesByDeployment: middleware.WithTracing(instancesByDeployment),
		Policies:              middleware.WithTracing(policies),
		TLSCertificates:       middleware.WithTracing(tlsCertificate),
	}, nil
}

package caches

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
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

	// dispatcher handles routing of invalidation events to all caches in this process.
	dispatcher *clustering.InvalidationDispatcher
}

// Close shuts down the caches and cleans up resources.
func (c *Caches) Close() error {
	if c.dispatcher != nil {
		return c.dispatcher.Close()
	}

	return nil
}

// Config defines the configuration options for initializing caches.
type Config struct {
	Clock clock.Clock

	// Broadcaster for distributed cache invalidation via gossip.
	// If nil, caches operate in local-only mode (no distributed invalidation).
	Broadcaster clustering.Broadcaster

	// NodeID identifies this node in the cluster (defaults to hostname-uniqueid to ensure uniqueness)
	NodeID string
}

// clusterOpts bundles the dispatcher and key converter functions needed for
// distributed cache invalidation.
type clusterOpts[K comparable] struct {
	dispatcher  *clustering.InvalidationDispatcher
	broadcaster clustering.Broadcaster
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
}

// createCache creates a cache instance with optional clustering support.
func createCache[K comparable, V any](
	cacheConfig cache.Config[K, V],
	opts *clusterOpts[K],
) (cache.Cache[K, V], error) {
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		return localCache, nil
	}

	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Broadcaster: opts.broadcaster,
		Dispatcher:  opts.dispatcher,
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
	if config.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		config.NodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	var dispatcher *clustering.InvalidationDispatcher
	var stringKeyOpts *clusterOpts[string]

	if config.Broadcaster != nil {
		var err error
		dispatcher, err = clustering.NewInvalidationDispatcher(config.Broadcaster)
		if err != nil {
			return nil, err
		}

		stringKeyOpts = &clusterOpts[string]{
			dispatcher:  dispatcher,
			broadcaster: config.Broadcaster,
			nodeID:      config.NodeID,
			keyToString: nil,
			stringToKey: nil,
		}
	}

	// Ensure the dispatcher is closed if any subsequent cache creation fails.
	initialized := false
	if dispatcher != nil {
		defer func() {
			if !initialized {
				_ = dispatcher.Close()
			}
		}()
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

	initialized = true
	return &Caches{
		FrontlineRoutes:       middleware.WithTracing(frontlineRoute),
		InstancesByDeployment: middleware.WithTracing(instancesByDeployment),
		Policies:              middleware.WithTracing(policies),
		TLSCertificates:       middleware.WithTracing(tlsCertificate),
		dispatcher:            dispatcher,
	}, nil
}

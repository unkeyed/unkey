package caches

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Caches holds all cache instances used throughout frontline.
type Caches struct {
	// HostName -> frontline Route
	FrontlineRoutes cache.Cache[string, db.FrontlineRoute]

	// EnvironmentID -> List of Sentinels
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]

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
		cache.Config[string, db.FrontlineRoute]{
			Fresh:    30 * time.Second,
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

	sentinelsByEnvironment, err := createCache(
		cache.Config[string, []db.Sentinel]{
			Fresh:    30 * time.Second,
			Stale:    2 * time.Minute,
			MaxSize:  10_000,
			Resource: "sentinels_by_environment",
			Clock:    config.Clock,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentinels by environment cache: %w", err)
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
		FrontlineRoutes:        middleware.WithTracing(frontlineRoute),
		SentinelsByEnvironment: middleware.WithTracing(sentinelsByEnvironment),
		TLSCertificates:        middleware.WithTracing(tlsCertificate),
		dispatcher:             dispatcher,
	}, nil
}

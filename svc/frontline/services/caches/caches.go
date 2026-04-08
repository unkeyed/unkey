package caches

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	batchmetrics "github.com/unkeyed/unkey/pkg/batch/metrics"
	buffermetrics "github.com/unkeyed/unkey/pkg/buffer/metrics"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	clusteringmetrics "github.com/unkeyed/unkey/pkg/cache/clustering/metrics"
	cachemetrics "github.com/unkeyed/unkey/pkg/cache/metrics"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// Caches holds all cache instances used throughout frontline.
type Caches struct {
	// HostName -> frontline Route
	FrontlineRoutes cache.Cache[string, db.FindFrontlineRouteByFQDNRow]

	// EnvironmentID -> List of Sentinels
	SentinelsByEnvironment cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]

	// DeploymentID -> List of Instances
	InstancesByDeployment cache.Cache[string, []db.FindInstancesByDeploymentIDRow]

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

	// CacheMetrics provides metrics for cache operations.
	CacheMetrics *cachemetrics.Metrics

	// ClusteringMetrics provides metrics for clustering operations.
	ClusteringMetrics *clusteringmetrics.Metrics

	// BatchMetrics provides metrics for batch operations in cluster cache invalidation.
	BatchMetrics *batchmetrics.Metrics

	// BufferMetrics provides metrics for buffer operations in cluster cache invalidation.
	BufferMetrics *buffermetrics.Metrics
}

// clusterOpts bundles the dispatcher and key converter functions needed for
// distributed cache invalidation.
type clusterOpts[K comparable] struct {
	dispatcher        *clustering.InvalidationDispatcher
	broadcaster       clustering.Broadcaster
	nodeID            string
	keyToString       func(K) string
	stringToKey       func(string) (K, error)
	clusteringMetrics *clusteringmetrics.Metrics
	batchMetrics      *batchmetrics.Metrics
	bufferMetrics     *buffermetrics.Metrics
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
		LocalCache:    localCache,
		Broadcaster:   opts.broadcaster,
		Dispatcher:    opts.dispatcher,
		Metrics:       opts.clusteringMetrics,
		BatchMetrics:  opts.batchMetrics,
		BufferMetrics: opts.bufferMetrics,
		NodeID:        opts.nodeID,
		KeyToString:   opts.keyToString,
		StringToKey:   opts.stringToKey,
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
		dispatcher, err = clustering.NewInvalidationDispatcher(config.Broadcaster, config.ClusteringMetrics)
		if err != nil {
			return nil, err
		}

		stringKeyOpts = &clusterOpts[string]{
			dispatcher:        dispatcher,
			broadcaster:       config.Broadcaster,
			nodeID:            config.NodeID,
			keyToString:       nil,
			stringToKey:       nil,
			clusteringMetrics: config.ClusteringMetrics,
			batchMetrics:      config.BatchMetrics,
			bufferMetrics:     config.BufferMetrics,
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
			Metrics:  config.CacheMetrics,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create frontline route cache: %w", err)
	}

	sentinelsByEnvironment, err := createCache(
		cache.Config[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]{
			Fresh:    5 * time.Second,
			Stale:    2 * time.Minute,
			MaxSize:  10_000,
			Resource: "sentinels_by_environment",
			Clock:    config.Clock,
			Metrics:  config.CacheMetrics,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentinels by environment cache: %w", err)
	}

	instancesByDeployment, err := createCache(
		cache.Config[string, []db.FindInstancesByDeploymentIDRow]{
			Fresh:    10 * time.Second,
			Stale:    60 * time.Second,
			MaxSize:  10_000,
			Resource: "instances_by_deployment",
			Clock:    config.Clock,
			Metrics:  config.CacheMetrics,
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
			Metrics:  config.CacheMetrics,
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
		InstancesByDeployment:  middleware.WithTracing(instancesByDeployment),
		TLSCertificates:        middleware.WithTracing(tlsCertificate),
		dispatcher:             dispatcher,
	}, nil
}

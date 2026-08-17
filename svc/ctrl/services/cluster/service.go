package cluster

import (
	"fmt"
	"time"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// notifiedReadyTTL is how long an entry in notifiedReady is kept before
// being eligible for cleanup. Five minutes is comfortably longer than any
// deployment's notify-ready window — once a deployment transitions to ready
// or terminal there's no value in remembering its entry.
const notifiedReadyTTL = 5 * time.Minute

// Infra certificate provisioning is confirmed once the acme_challenges row
// exists, and that state is effectively permanent, so we cache it aggressively
// to keep the steady-state heartbeat path off the DB. The window only bounds
// re-verification of an already-provisioned domain; a failed attempt is never
// cached, so it keeps retrying on every heartbeat regardless.
const (
	infraCertCacheTTL     = 1 * time.Hour
	infraCertCacheMaxSize = 256
)

// Populated cluster identities are immutable. Caching the complete identity
// avoids a database read on every cluster-scoped RPC without allowing one cell
// to replace another through Heartbeat.
const (
	clusterCacheFresh   = 5 * time.Minute
	clusterCacheStale   = 15 * time.Minute
	clusterCacheMaxSize = 256
)

// clusterCacheKey is the complete immutable identity supplied by Krane.
type clusterCacheKey struct {
	cellID   string
	platform string
	region   string
}

// Service implements [ctrlv1connect.ClusterServiceHandler] to synchronize desired state
// between the control plane and krane agents. It provides streaming RPCs for watching
// deployment changes, point queries for fetching individual resource states,
// and status reporting endpoints for agents to report observed state back to the control plane.
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db      db.Database
	restate *ingress.Client
	bearer  string
	// notifiedReady dedups Restate NotifyInstancesReady calls so we don't
	// fire on every krane status report once the threshold is met. Keys
	// are "deployment:<id>".
	notifiedReady *expiringSet[string]
	// clusterCache memoizes immutable cluster identities for cluster-scoped RPCs.
	clusterCache cache.Cache[clusterCacheKey, db.FindClusterRow]
	// topologyCache caches FindDeploymentTopologyMinReplicas lookups
	// keyed by deployment_id. Topology is written once at deploy time,
	// then read on every instance status report, so caching removes an
	// RO hit from the notify path. Empty results are not cached (see
	// findTopologyMinReplicas) because a missed race against the
	// topology write must be retried, not sealed in.
	topologyCache cache.Cache[string, []db.FindDeploymentTopologyMinReplicasRow]
	// instanceEvents buffers container lifecycle events from krane before
	// flushing them to ClickHouse. Always non-nil — if ClickHouse isn't
	// configured for the api process this is a noop processor.
	instanceEvents *batch.BatchProcessor[schema.InstanceEventV1]
	// regionalDomain is the base domain for per-region wildcard certificates
	// (*.{region}.{platform}.{regionalDomain}). When a new region registers via
	// Heartbeat, EnsureInfraCertificate provisions its wildcard cert. Empty
	// disables region cert issuance (e.g. local dev with no ACME configured).
	regionalDomain string
	// provisionedCerts caches infra domains whose certificate records already
	// exist so EnsureInfraCertificate skips its DB check on the steady-state
	// heartbeat path. Only confirmed-provisioned domains are stored, so a failed
	// attempt is never cached and keeps retrying.
	provisionedCerts cache.Cache[string, bool]
}

// Config holds the configuration for creating a new cluster [Service].
type Config struct {
	// Database provides read and write access for querying and updating resource state.
	Database db.Database

	// Restate is the ingress client used to trigger durable workflows.
	Restate *ingress.Client

	// Bearer is the authentication token that agents must provide in the Authorization header.
	Bearer string

	// Clock backs cache freshness accounting. When nil, a real-time clock is used.
	Clock clock.Clock

	// TopologyCache backs FindDeploymentTopologyMinReplicas lookups on
	// the notify-ready path. Required.
	TopologyCache cache.Cache[string, []db.FindDeploymentTopologyMinReplicasRow]

	// InstanceEvents is the batch processor that absorbs container
	// lifecycle events for ClickHouse ingestion. Required — pass a noop
	// (batch.NewNoop) when ClickHouse is unavailable.
	InstanceEvents *batch.BatchProcessor[schema.InstanceEventV1]

	// RegionalDomain is the base domain for per-region wildcard certificates
	// (*.{region}.{platform}.{RegionalDomain}). Empty disables automatic
	// region certificate issuance on Heartbeat.
	RegionalDomain string
}

// New creates a new cluster [Service] with the given configuration. The returned service
// is ready to be registered with a Connect server. A background sweeper is
// started that periodically drops stale entries from notifiedReady so the
// set doesn't grow unbounded.
func New(cfg Config) (*Service, error) {
	// Required dependencies are validated up-front so misconfiguration
	// fails the boot loud instead of nil-panicking on the first event.
	if cfg.TopologyCache == nil {
		return nil, fmt.Errorf("cluster: TopologyCache is required")
	}
	if cfg.InstanceEvents == nil {
		return nil, fmt.Errorf("cluster: InstanceEvents is required (use batch.NewNoop when ClickHouse is unavailable)")
	}

	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}
	clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{
		Fresh:    clusterCacheFresh,
		Stale:    clusterCacheStale,
		MaxSize:  clusterCacheMaxSize,
		Resource: "ctrl_clusters",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster cache: %w", err)
	}
	provisionedCerts, err := cache.New(cache.Config[string, bool]{
		Fresh:    infraCertCacheTTL,
		Stale:    infraCertCacheTTL,
		MaxSize:  infraCertCacheMaxSize,
		Resource: "ctrl_infra_certs",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create infra cert cache: %w", err)
	}
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		bearer:                             cfg.Bearer,
		notifiedReady:                      newExpiringSet[string](notifiedReadyTTL),
		clusterCache:                       clusterCache,
		topologyCache:                      cfg.TopologyCache,
		instanceEvents:                     cfg.InstanceEvents,
		regionalDomain:                     cfg.RegionalDomain,
		provisionedCerts:                   provisionedCerts,
	}
	repeat.Every(notifiedReadyTTL, func() {
		if dropped := s.notifiedReady.Sweep(); dropped > 0 {
			logger.Info("swept stale notifiedReady entries", "dropped", dropped)
		}
	})
	return s, nil
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)

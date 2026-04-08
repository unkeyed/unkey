package router

import (
	"context"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	batchmetrics "github.com/unkeyed/unkey/pkg/batch/metrics"
	buffermetrics "github.com/unkeyed/unkey/pkg/buffer/metrics"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	clusteringmetrics "github.com/unkeyed/unkey/pkg/cache/clustering/metrics"
	cachemetrics "github.com/unkeyed/unkey/pkg/cache/metrics"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
)

type Service interface {
	GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error)
	SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error)
	GetPolicies(ctx context.Context, deployment db.Deployment) ([]*sentinelv1.Policy, error)
}

type Config struct {
	DB            db.Database
	Clock         clock.Clock
	EnvironmentID string
	Platform      string
	Region        string

	// Broadcaster for distributed cache invalidation via gossip.
	// If nil, caches operate in local-only mode (no distributed invalidation).
	Broadcaster clustering.Broadcaster

	// NodeID identifies this node in the cluster
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

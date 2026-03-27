package router

import (
	"context"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
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
}

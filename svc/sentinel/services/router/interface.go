package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/sentinel/internal/db"
)

type Service interface {
	GetDeployment(ctx context.Context, deploymentID string) (db.FindDeploymentByIdRow, error)
	SelectInstance(ctx context.Context, deploymentID string) (db.FindInstancesByDeploymentIdAndRegionIDRow, error)
}

type Config struct {
	DB            db.Querier
	Clock         clock.Clock
	EnvironmentID string
	Platform      string
	Region        string
	RegionID      string

	// Broadcaster for distributed cache invalidation via gossip.
	// If nil, caches operate in local-only mode (no distributed invalidation).
	Broadcaster clustering.Broadcaster

	// NodeID identifies this node in the cluster
	NodeID string
}

package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type RouteDecision struct {
	// LocalSentinel is set if there's a healthy sentinel in the local region
	LocalSentinel *db.Sentinel

	// NearestNLBRegion is set if we need to forward to another region's NLB
	NearestNLBRegion string

	// DeploymentID to pass in X-Unkey-Deployment-Id header
	DeploymentID string
}

type Service interface {
	LookupByHostname(ctx context.Context, hostname string) (*db.FrontlineRoute, []db.Sentinel, error)
	SelectSentinel(route *db.FrontlineRoute, sentinels []db.Sentinel) (*RouteDecision, error)
}

type Config struct {
	Logger                 logging.Logger
	Region                 string
	DB                     db.Database
	FrontlineRouteCache    cache.Cache[string, db.FrontlineRoute]
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]
}

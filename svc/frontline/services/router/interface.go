package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

type RouteDecision struct {
	// LocalSentinel is set if there's a healthy sentinel in the local region
	LocalSentinel *db.Sentinel

	// NearestNLBRegionPlatform is set if we need to forward to another region's NLB
	// The format must be dns compatible and is typically "<region>.<platform>"
	NearestNLBRegionPlatform string

	// DeploymentID to pass in X-Unkey-Deployment-Id header
	DeploymentID string
}

type Service interface {
	LookupByHostname(ctx context.Context, hostname string) (*db.FrontlineRoute, []db.Sentinel, error)
	SelectSentinel(route *db.FrontlineRoute, sentinels []db.Sentinel) (*RouteDecision, error)
}

type Config struct {
	Platform               string
	Region                 string
	DB                     db.Database
	FrontlineRouteCache    cache.Cache[string, db.FrontlineRoute]
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]
}

package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type RouteDecision struct {
	// LocalSentinelAddress is set if there's a healthy sentinel in the local region.
	LocalSentinelAddress string

	// NearestNLBRegionPlatform is set if we need to forward to another region's NLB
	// The format must be dns compatible and is typically "<region>.<platform>"
	NearestNLBRegionPlatform string

	// DeploymentID to pass in X-Unkey-Deployment-Id header
	DeploymentID string
}

type Service interface {
	LookupByHostname(ctx context.Context, hostname string) (*db.FindFrontlineRouteByFQDNRow, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error)
	SelectSentinel(route *db.FindFrontlineRouteByFQDNRow, sentinels []db.FindHealthyRoutableSentinelsByEnvironmentIDRow) (*RouteDecision, error)
}

type Config struct {
	Platform               string
	Region                 string
	DB                     db.Querier
	FrontlineRouteCache    cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	SentinelsByEnvironment cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
}

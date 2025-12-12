package router

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
	LookupByHostname(ctx context.Context, hostname string) (*db.IngressRoute, []db.Sentinel, error)
	SelectSentinel(route *db.IngressRoute, sentinels []db.Sentinel) (*RouteDecision, error)
}

type Config struct {
	Logger                 logging.Logger
	Region                 string
	DB                     db.Database
	IngressRouteCache      cache.Cache[string, db.IngressRoute]
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]
}

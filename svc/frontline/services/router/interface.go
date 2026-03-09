package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// RouteDecision contains the routing decision for a request.
type RouteDecision struct {
	DeploymentID string

	// Exactly one of these is set:
	// LocalSentinelAddress is the K8s address for h2c forwarding to a local sentinel.
	LocalSentinelAddress string
	// RemoteRegionPlatform is the "region.platform" string for NLB forwarding.
	RemoteRegionPlatform string
}

// IsLocal returns true if the request should be forwarded to a local sentinel.
func (d *RouteDecision) IsLocal() bool {
	return d.LocalSentinelAddress != ""
}

type Service interface {
	// Route determines where to forward a request based on hostname.
	// Handles hostname lookup, sentinel discovery, instance verification, and region selection.
	Route(ctx context.Context, hostname string) (*RouteDecision, error)

	// ValidateHostname checks if a hostname has a configured frontline route.
	// Used by ACME handler to verify domain ownership.
	ValidateHostname(ctx context.Context, hostname string) error
}

type Config struct {
	Platform               string
	Region                 string
	DB                     db.Querier
	FrontlineRouteCache    cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	SentinelsByEnvironment cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
	InstancesByDeployment  cache.Cache[string, []db.Instance]
}

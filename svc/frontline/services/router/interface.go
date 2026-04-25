package router

import (
	"context"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type Destination string

const (
	DestinationLocalSentinel Destination = "sentinel"
	DestinationRemoteRegion  Destination = "region"
)

type RouteDecision struct {
	DeploymentID string
	Destination  Destination
	// Address is the K8s sentinel address (local) or "region.platform" string (remote).
	Address string
	// Redirects are the per-FQDN edge-redirect rules parsed from
	// frontline_routes.edge_redirect_config. Empty for routes with no
	// rules (the common case for auto-generated preview URLs).
	Redirects []*edgeredirectv1.Rule
}

type Service interface {
	// Route determines where to forward a request based on hostname.
	Route(ctx context.Context, hostname string) (RouteDecision, error)

	// ValidateHostname checks if a hostname has a configured frontline route.
	ValidateHostname(ctx context.Context, hostname string) error
}

type Config struct {
	Platform               string
	Region                 string
	DB                     db.Querier
	FrontlineRouteCache    cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	SentinelsByEnvironment cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
	InstancesByDeployment  cache.Cache[string, []db.FindInstancesByDeploymentIDRow]
}

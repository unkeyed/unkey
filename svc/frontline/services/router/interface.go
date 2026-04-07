package router

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type Destination string

const (
	DestinationLocalSentinel Destination = "sentinel"
	DestinationRemoteRegion  Destination = "region"
	DestinationPortal        Destination = "portal"
)

// RouteDecision describes where frontline should forward a request.
//
// For deployment routes (DestinationLocalSentinel / DestinationRemoteRegion),
// DeploymentID and Address are populated.
//
// For portal routes (DestinationPortal), PathPrefix is populated and Address
// holds the portal service address from config.
type RouteDecision struct {
	DeploymentID string
	Destination  Destination
	// Address is the K8s sentinel address (local), "region.platform" string (remote),
	// or portal service address (portal).
	Address string
	// PathPrefix is set only for portal routes (e.g. "/portal").
	PathPrefix string
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
	InstancesByDeployment  cache.Cache[string, []db.Instance]
	// PortalAddr is the address of the portal service (e.g. "portal:3000").
	// When empty, portal routes return 503.
	PortalAddr string
}

package router

import (
	"context"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type Destination string

const (
	// DestinationLocalInstance routes to a deployment instance running in
	// this region — the merged frontline runs the engine and proxies
	// directly to the instance.
	DestinationLocalInstance Destination = "instance"
	// DestinationRemoteRegion forwards to a peer frontline in another
	// region. The peer redoes the full hostname → engine → instance chain.
	DestinationRemoteRegion Destination = "region"
)

// RouteDecision is the output of Route. For local instance routing it carries
// the deployment, the resolved instance, the parsed policies, and the
// upstream protocol the proxy uses to pick a transport. For cross-region
// forwarding only Address is populated.
type RouteDecision struct {
	Destination      Destination
	DeploymentID     string
	EnvironmentID    string
	WorkspaceID      string
	ProjectID        string
	UpstreamProtocol db.DeploymentsUpstreamProtocol
	Instance         db.FindInstancesByDeploymentIDRow
	Policies         []*frontlinev1.Policy
	// Address is the running instance address (local) or "region.platform"
	// peer-frontline target (remote).
	Address string
}

type Service interface {
	// Route determines where to forward a request based on hostname and,
	// for local routing, returns the parsed policies and chosen instance.
	Route(ctx context.Context, hostname string) (RouteDecision, error)

	// ValidateHostname checks if a hostname has a configured frontline route.
	ValidateHostname(ctx context.Context, hostname string) error
}

type Config struct {
	Platform              string
	Region                string
	DB                    db.Querier
	FrontlineRouteCache   cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	InstancesByDeployment cache.Cache[string, []db.FindInstancesByDeploymentIDRow]
	PolicyCache           cache.Cache[string, []*frontlinev1.Policy]
}

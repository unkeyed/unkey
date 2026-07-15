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

// RouteDecision is the output of Route.
//
// For DestinationLocalInstance: LocalInstances carries the candidate pods
// in shuffled order — the caller attempts them sequentially, advancing on
// dial failures. RemoteRegionAddress, when non-empty, is a standby peer
// region for the caller to fall through to once every local instance has
// dial-failed; empty when this is the only region with running instances.
//
// For DestinationRemoteRegion: only RemoteRegionAddress is populated.
// LocalInstances is empty.
type RouteDecision struct {
	Destination         Destination
	DeploymentID        string
	EnvironmentID       string
	WorkspaceID         string
	ProjectID           string
	AppID               string
	LocalInstances      []db.FindInstancesByDeploymentIDRow
	RemoteRegionAddress string
	UpstreamProtocol    db.DeploymentsUpstreamProtocol
	Policies            []*frontlinev1.Policy
}

type Service interface {
	// Route resolves a hostname to a forwarding decision. For local routes
	// the returned decision includes the candidate instances in the order
	// the caller should try them, plus the parsed policies to evaluate
	// before forwarding.
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

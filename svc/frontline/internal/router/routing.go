package router

import (
	"fmt"
	"math/rand/v2"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

var regionProximity = map[string][]string{
	// US East
	"us-east-1.aws": {"us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ca-central-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},
	"us-east-2.aws": {"us-east-1.aws", "us-west-2.aws", "us-west-1.aws", "ca-central-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},

	// US West
	"us-west-1.aws": {"us-west-2.aws", "us-east-2.aws", "us-east-1.aws", "ca-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"us-west-2.aws": {"us-west-1.aws", "us-east-2.aws", "us-east-1.aws", "ca-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws", "eu-west-1.aws", "eu-central-1.aws"},

	// Canada
	"ca-central-1.aws": {"us-east-2.aws", "us-east-1.aws", "us-west-2.aws", "us-west-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},

	// Europe
	"eu-west-1.aws":    {"eu-west-2.aws", "eu-central-1.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-west-2.aws":    {"eu-west-1.aws", "eu-central-1.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-central-1.aws": {"eu-west-1.aws", "eu-west-2.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-north-1.aws":   {"eu-central-1.aws", "eu-west-1.aws", "eu-west-2.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},

	// Asia Pacific
	"ap-south-1.aws":     {"ap-southeast-1.aws", "ap-southeast-2.aws", "ap-northeast-1.aws", "eu-west-1.aws", "eu-central-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws"},
	"ap-northeast-1.aws": {"ap-southeast-1.aws", "ap-southeast-2.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"ap-southeast-1.aws": {"ap-southeast-2.aws", "ap-northeast-1.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"ap-southeast-2.aws": {"ap-southeast-1.aws", "ap-northeast-1.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},

	// Local development
	"local.dev": {},
}

// selectDestination decides whether to run the engine + proxy locally or
// forward to a peer frontline.
//
// If any running instance exists in this region, the returned decision is
// local and carries every local instance in shuffled order — the caller
// walks the list, trying the next on dial failures. The decision also
// carries the nearest peer region (if one exists) as RemoteRegionAddress,
// which the caller falls through to once every local instance has dial-
// failed. Empty when no other region has running instances.
//
// Otherwise (no local instances), the decision points at the nearest peer
// region that has at least one running instance, and LocalInstances is
// empty.
func (s *service) selectDestination(
	route db.FindFrontlineRouteByFQDNRow,
	instances []db.FindInstancesByDeploymentIDRow,
	policies []*frontlinev1.Policy,
) (RouteDecision, error) {
	if len(instances) == 0 {
		return RouteDecision{}, fault.New("no instances",
			fault.Code(codes.Frontline.Routing.NoDeploymentInstances.URN()),
			fault.Internal(fmt.Sprintf("no instances for deployment %s", route.DeploymentID)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	var localRunning []db.FindInstancesByDeploymentIDRow
	regionsWithInstance := make(map[string]bool)
	for _, inst := range instances {
		if inst.Status != db.InstancesStatusRunning {
			continue
		}
		key := fmt.Sprintf("%s.%s", inst.RegionName, inst.RegionPlatform)
		regionsWithInstance[key] = true
		if key == s.regionPlatform {
			localRunning = append(localRunning, inst)
		}
	}

	if len(regionsWithInstance) == 0 {
		return RouteDecision{}, fault.New("no running instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s", route.DeploymentID)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if len(localRunning) > 0 {
		rand.Shuffle(len(localRunning), func(i int, j int) {
			localRunning[i], localRunning[j] = localRunning[j], localRunning[i]
		})
		// Pick a standby peer region for the handler to fall through to
		// if every local instance dial-fails. Empty when this is the only
		// region with running instances — in that case there is nowhere
		// to fall through to and the dial failures surface to the client.
		return RouteDecision{
			Destination:         DestinationLocalInstance,
			DeploymentID:        route.DeploymentID,
			EnvironmentID:       route.EnvironmentID,
			WorkspaceID:         localRunning[0].WorkspaceID,
			ProjectID:           localRunning[0].ProjectID,
			UpstreamProtocol:    route.UpstreamProtocol,
			Policies:            policies,
			LocalInstances:      localRunning,
			RemoteRegionAddress: s.findNearestRegionPlatform(regionsWithInstance),
		}, nil
	}

	nearestRegion := s.findNearestRegionPlatform(regionsWithInstance)
	if nearestRegion == "" {
		return RouteDecision{}, fault.New("no reachable region from "+s.regionPlatform,
			fault.Code(codes.Frontline.Routing.NoReachableRegion.URN()),
			fault.Internal("running instances exist but no region is reachable"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	//nolint:exhaustruct
	return RouteDecision{
		Destination:         DestinationRemoteRegion,
		DeploymentID:        route.DeploymentID,
		EnvironmentID:       route.EnvironmentID,
		RemoteRegionAddress: nearestRegion,
	}, nil
}

// findNearestRegionPlatform returns the peer region (other than the local
// one) most likely to be reachable, given the set of regions known to have
// running instances for the deployment. Returns "" if no peer is available.
//
// The proximity table is the primary source of order. When the local
// region is missing from the table, or the proximity list does not name
// any region in regionsWithInstance (the table is incomplete in places —
// e.g. us-east-1.aws does not list eu-north-1.aws), we fall back to map-
// iteration order. Iteration order is non-deterministic but at least
// returns *something* reachable instead of failing closed.
func (s *service) findNearestRegionPlatform(regionsWithInstance map[string]bool) string {
	if proximityList, exists := regionProximity[s.regionPlatform]; exists {
		for _, region := range proximityList {
			if regionsWithInstance[region] {
				return region
			}
		}
	}

	for region := range regionsWithInstance {
		if region != s.regionPlatform {
			return region
		}
	}

	return ""
}

package router

import (
	"fmt"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/array"
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
// forward to a peer frontline. The choice keys off "is there a running
// instance for this deployment in this region?" — if yes, route locally;
// otherwise forward to the nearest region that has one.
func (s *service) selectDestination(
	route db.FindFrontlineRouteByFQDNRow,
	instances []db.FindInstancesByDeploymentIDRow,
	policies []*frontlinev1.Policy,
) (RouteDecision, error) {
	if len(instances) == 0 {
		routingErrorsTotal.WithLabelValues("no_instances").Inc()
		return RouteDecision{}, fault.New("no instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
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
		routingErrorsTotal.WithLabelValues("no_running_instances").Inc()
		return RouteDecision{}, fault.New("no running instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s", route.DeploymentID)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if len(localRunning) > 0 {
		selected := array.Random(localRunning)
		return RouteDecision{
			Destination:      DestinationLocalInstance,
			DeploymentID:     route.DeploymentID,
			EnvironmentID:    route.EnvironmentID,
			WorkspaceID:      selected.WorkspaceID,
			ProjectID:        selected.ProjectID,
			UpstreamProtocol: route.UpstreamProtocol,
			Instance:         selected,
			Policies:         policies,
			Address:          selected.Address,
		}, nil
	}

	nearestRegion := s.findNearestRegionPlatform(regionsWithInstance)
	if nearestRegion == "" {
		routingErrorsTotal.WithLabelValues("no_reachable_region").Inc()
		return RouteDecision{}, fault.New("no reachable region from "+s.regionPlatform,
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("running instances exist but no region is reachable"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	//nolint:exhaustruct
	return RouteDecision{
		Destination:   DestinationRemoteRegion,
		DeploymentID:  route.DeploymentID,
		EnvironmentID: route.EnvironmentID,
		Address:       nearestRegion,
	}, nil
}

func (s *service) findNearestRegionPlatform(regionsWithInstance map[string]bool) string {
	proximityList, exists := regionProximity[s.regionPlatform]
	if exists {
		for _, region := range proximityList {
			if regionsWithInstance[region] {
				return region
			}
		}
	}

	for region := range regionsWithInstance {
		return region
	}

	return ""
}

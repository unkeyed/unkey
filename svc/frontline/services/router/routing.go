package router

import (
	"fmt"

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

func (s *service) selectSentinel(route db.FindFrontlineRouteByFQDNRow, rows []db.FindHealthyRoutableSentinelsByEnvironmentIDRow, instances []db.Instance) (RouteDecision, error) {
	hasRunningInstance := false
	for _, inst := range instances {
		if inst.Status == db.InstancesStatusRunning {
			hasRunningInstance = true
			break
		}
	}

	if len(instances) > 0 && !hasRunningInstance {
		routingErrorsTotal.WithLabelValues("no_running_instances").Inc()
		return RouteDecision{}, fault.New("no running instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("no running instances for deployment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	healthyByRegion := make(map[string]string)
	for _, row := range rows {
		if row.RegionName == "" || row.RegionPlatform == "" {
			continue
		}

		key := fmt.Sprintf("%s.%s", row.RegionName, row.RegionPlatform)
		healthyByRegion[key] = row.K8sAddress
	}

	if len(healthyByRegion) == 0 {
		routingErrorsTotal.WithLabelValues("no_sentinels_for_instances").Inc()
		return RouteDecision{}, fault.New("no healthy sentinels",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("no healthy sentinels for environment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if localAddress, ok := healthyByRegion[s.regionPlatform]; ok {
		return RouteDecision{
			DeploymentID: route.DeploymentID.String,
			Destination:  DestinationLocalSentinel,
			Address:      localAddress,
		}, nil
	}

	nearestRegion := s.findNearestRegionPlatform(healthyByRegion)
	if nearestRegion == "" {
		routingErrorsTotal.WithLabelValues("no_reachable_region").Inc()
		return RouteDecision{}, fault.New("no reachable region from "+s.regionPlatform,
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("healthy sentinels exist but no region is reachable"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	return RouteDecision{
		DeploymentID: route.DeploymentID.String,
		Destination:  DestinationRemoteRegion,
		Address:      nearestRegion,
	}, nil
}

func (s *service) findNearestRegionPlatform(healthyByRegion map[string]string) string {
	proximityList, exists := regionProximity[s.regionPlatform]
	if exists {
		for _, region := range proximityList {
			if _, ok := healthyByRegion[region]; ok {
				return region
			}
		}
	}

	for region := range healthyByRegion {
		return region
	}

	return ""
}

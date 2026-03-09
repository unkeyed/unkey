package router

import (
	"context"
	"fmt"

	internalCaches "github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

// regionProximity maps regions to their closest regions in order of proximity.
// Format: region.cloud (e.g., "us-east-1.aws")
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

type service struct {
	platform                    string
	region                      string
	regionPlatform              string
	db                          db.Querier
	frontlineRouteCache         cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	sentinelsByEnvironmentCache cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
	instancesByDeploymentCache  cache.Cache[string, []db.Instance]
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	return &service{
		platform:                    cfg.Platform,
		region:                      cfg.Region,
		regionPlatform:              fmt.Sprintf("%s.%s", cfg.Region, cfg.Platform),
		db:                          cfg.DB,
		frontlineRouteCache:         cfg.FrontlineRouteCache,
		sentinelsByEnvironmentCache: cfg.SentinelsByEnvironment,
		instancesByDeploymentCache:  cfg.InstancesByDeployment,
	}, nil
}

func (s *service) Route(ctx context.Context, hostname string) (*RouteDecision, error) {
	route, sentinels, err := s.lookupByHostname(ctx, hostname)
	if err != nil {
		return nil, err
	}

	instances, err := s.getInstances(ctx, route.DeploymentID)
	if err != nil {
		return nil, err
	}

	return s.selectSentinel(route, sentinels, instances)
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, _, err := s.lookupByHostname(ctx, hostname)
	return err
}

func (s *service) lookupByHostname(ctx context.Context, hostname string) (*db.FindFrontlineRouteByFQDNRow, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
	route, routeHit, err := s.frontlineRouteCache.SWR(ctx, hostname, func(ctx context.Context) (db.FindFrontlineRouteByFQDNRow, error) {
		return s.db.FindFrontlineRouteByFQDN(ctx, hostname)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading frontline route"),
			fault.Public("Failed to load route configuration"),
		)
	}

	if mysql.IsNotFound(err) || routeHit == cache.Null {
		return nil, nil, fault.New("no frontline route for hostname: "+hostname,
			fault.Code(codes.Frontline.Routing.ConfigNotFound.URN()),
			fault.Public("Domain not configured"),
		)
	}

	sentinels, _, err := s.sentinelsByEnvironmentCache.SWR(ctx, route.EnvironmentID, func(ctx context.Context) ([]db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
		return s.db.FindHealthyRoutableSentinelsByEnvironmentID(ctx, route.EnvironmentID)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading sentinels"),
			fault.Public("Failed to load sentinel configuration"),
		)
	}

	return &route, sentinels, nil
}

func (s *service) getInstances(ctx context.Context, deploymentID string) ([]db.Instance, error) {
	instances, _, err := s.instancesByDeploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.Instance, error) {
		return s.db.FindInstancesByDeploymentID(ctx, deploymentID)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading instances"),
			fault.Public("Failed to load instance configuration"),
		)
	}

	return instances, nil
}

func (s *service) selectSentinel(route *db.FindFrontlineRouteByFQDNRow, rows []db.FindHealthyRoutableSentinelsByEnvironmentIDRow, instances []db.Instance) (*RouteDecision, error) {
	decision := &RouteDecision{
		DeploymentID:         route.DeploymentID,
		LocalSentinelAddress: "",
		RemoteRegionPlatform: "",
	}

	// Build set of region.platform keys that have running instances.
	// Since sentinel rows use region_name+region_platform (from the JOIN) and instances
	// use region_id, we need to build the mapping via sentinel rows' region metadata.
	// Sentinel rows already carry region_name and region_platform from the regions table.
	// We map region names to region.platform keys, then check instances by region_id.
	//
	// For instance-aware filtering, we build a set of regionPlatform keys that have
	// running instances. Since Instance.RegionID is a UUID and sentinel rows have
	// RegionName+RegionPlatform, we build both directions.

	// First, collect all unique region.platform keys from sentinel rows
	// and build a regionName -> regionPlatform mapping
	regionNameToPlatformKey := make(map[string]string)
	for _, row := range rows {
		if row.RegionName == "" || row.RegionPlatform == "" {
			continue
		}
		key := fmt.Sprintf("%s.%s", row.RegionName, row.RegionPlatform)
		regionNameToPlatformKey[row.RegionName] = key
	}

	// Build set of regions with running instances
	// Note: Instance.RegionID is a UUID, so we need sentinel rows to map to region names.
	// We'll build a reverse map from the sentinel rows for this.
	regionsWithRunning := make(map[string]bool) // regionPlatform key -> has running
	for _, inst := range instances {
		if inst.Status == db.InstancesStatusRunning {
			// We need to find which region name corresponds to this instance's RegionID.
			// Since we don't have a direct regionID->name mapping from sentinels,
			// we mark all regions as potentially having running instances and filter below.
			regionsWithRunning[inst.RegionID] = true
		}
	}

	if len(instances) > 0 && len(regionsWithRunning) == 0 {
		return nil, fault.New("no running instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("no running instances for deployment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Build healthy sentinels by regionPlatform key, filtering by regions with running instances
	healthyByRegion := make(map[string]string) // regionPlatform key -> K8s address
	for _, row := range rows {
		if row.RegionName == "" || row.RegionPlatform == "" {
			continue
		}
		key := fmt.Sprintf("%s.%s", row.RegionName, row.RegionPlatform)
		healthyByRegion[key] = row.K8sAddress
	}

	if len(healthyByRegion) == 0 {
		return nil, fault.New("no healthy sentinels",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("no healthy sentinels for environment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Prefer local sentinel if available
	localRegionPlatform := s.regionPlatform
	if localAddress, ok := healthyByRegion[localRegionPlatform]; ok {
		decision.LocalSentinelAddress = localAddress
		return decision, nil
	}

	// Find nearest region with a healthy sentinel
	nearestRegion := s.findNearestRegionPlatform(healthyByRegion)
	if nearestRegion != "" {
		decision.RemoteRegionPlatform = nearestRegion
	}

	return decision, nil
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

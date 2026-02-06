package router

import (
	"context"

	internalCaches "github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
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
	region                      string
	db                          db.Database
	frontlineRouteCache         cache.Cache[string, db.FrontlineRoute]
	sentinelsByEnvironmentCache cache.Cache[string, []db.Sentinel]
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	return &service{
		region:                      cfg.Region,
		db:                          cfg.DB,
		frontlineRouteCache:         cfg.FrontlineRouteCache,
		sentinelsByEnvironmentCache: cfg.SentinelsByEnvironment,
	}, nil
}

func (s *service) LookupByHostname(ctx context.Context, hostname string) (*db.FrontlineRoute, []db.Sentinel, error) {
	route, routeHit, err := s.frontlineRouteCache.SWR(ctx, hostname, func(ctx context.Context) (db.FrontlineRoute, error) {
		return db.Query.FindFrontlineRouteByFQDN(ctx, s.db.RO(), hostname)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading frontline route"),
			fault.Public("Failed to load route configuration"),
		)
	}

	if db.IsNotFound(err) || routeHit == cache.Null {
		return nil, nil, fault.New("no frontline route for hostname: "+hostname,
			fault.Code(codes.Frontline.Routing.ConfigNotFound.URN()),
			fault.Public("Domain not configured"),
		)
	}

	sentinels, _, err := s.sentinelsByEnvironmentCache.SWR(ctx, route.EnvironmentID, func(ctx context.Context) ([]db.Sentinel, error) {
		return db.Query.FindSentinelsByEnvironmentID(ctx, s.db.RO(), route.EnvironmentID)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading sentinels"),
			fault.Public("Failed to load sentinel configuration"),
		)
	}

	return &route, sentinels, nil
}

func (s *service) SelectSentinel(route *db.FrontlineRoute, sentinels []db.Sentinel) (*RouteDecision, error) {
	decision := &RouteDecision{
		DeploymentID:     route.DeploymentID,
		LocalSentinel:    nil,
		NearestNLBRegion: "",
	}

	healthyByRegion := make(map[string]*db.Sentinel)
	for i := range sentinels {
		gw := &sentinels[i]
		if gw.Health != db.SentinelsHealthHealthy {
			continue
		}

		healthyByRegion[gw.Region] = gw
	}

	if len(healthyByRegion) == 0 {
		return nil, fault.New("no healthy sentinels",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("no healthy sentinels for environment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if localGw, ok := healthyByRegion[s.region]; ok {
		decision.LocalSentinel = localGw
		return decision, nil
	}

	nearestRegion := s.findNearestRegion(healthyByRegion)
	if nearestRegion != "" {
		decision.NearestNLBRegion = nearestRegion
	}

	return decision, nil
}

func (s *service) findNearestRegion(healthyByRegion map[string]*db.Sentinel) string {
	proximityList, exists := regionProximity[s.region]
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

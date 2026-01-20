package router

import (
	"context"

	internalCaches "github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/frontline/services/resilience"
	"go.opentelemetry.io/otel/attribute"
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
	logger                           logging.Logger
	region                           string
	db                               db.Database
	clock                            clock.Clock
	frontlineRouteCache              cache.Cache[string, db.FrontlineRoute]
	sentinelsByEnvironmentCache      cache.Cache[string, []db.Sentinel]
	runningInstanceRegionsByDeployID cache.Cache[string, []string]
	resilienceTracker                resilience.Tracker
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}
	return &service{
		logger:                           cfg.Logger,
		region:                           cfg.Region,
		db:                               cfg.DB,
		clock:                            clk,
		frontlineRouteCache:              cfg.FrontlineRouteCache,
		sentinelsByEnvironmentCache:      cfg.SentinelsByEnvironment,
		runningInstanceRegionsByDeployID: cfg.RunningInstanceRegionsByDeploy,
		resilienceTracker:                cfg.ResilienceTracker,
	}, nil
}

func (s *service) LookupByHostname(ctx context.Context, hostname string) (*db.FrontlineRoute, []db.Sentinel, error) {
	ctx, span := tracing.Start(ctx, "router.lookup_hostname")
	defer span.End()
	span.SetAttributes(attribute.String("hostname", hostname))

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

func (s *service) SelectSentinel(ctx context.Context, route *db.FrontlineRoute, sentinels []db.Sentinel) (*RouteDecision, error) {
	ctx, span := tracing.Start(ctx, "router.select_sentinel")
	defer span.End()
	span.SetAttributes(
		attribute.String("deployment_id", route.DeploymentID),
		attribute.String("environment_id", route.EnvironmentID),
		attribute.Int("sentinel_count", len(sentinels)),
	)

	decision := &RouteDecision{
		DeploymentID:     route.DeploymentID,
		LocalSentinel:    nil,
		NearestNLBRegion: "",
	}

	runningRegions, _, err := s.runningInstanceRegionsByDeployID.SWR(ctx, route.DeploymentID, func(ctx context.Context) ([]string, error) {
		return db.Query.FindRunningInstanceRegionsByDeploymentID(ctx, s.db.RO(), route.DeploymentID)
	}, internalCaches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading running instance regions"),
			fault.Public("Failed to check instance availability"),
		)
	}

	runningRegionsSet := make(map[string]bool, len(runningRegions))
	for _, region := range runningRegions {
		runningRegionsSet[region] = true
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

	eligibleByRegion := make(map[string]*db.Sentinel)
	for region, sentinel := range healthyByRegion {
		if runningRegionsSet[region] {
			eligibleByRegion[region] = sentinel
		}
	}

	if len(eligibleByRegion) == 0 {
		return nil, fault.New("no regions with running instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("healthy sentinels exist but no regions have running instances"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	now := s.clock.Now()
	if s.resilienceTracker != nil {
		filtered := make(map[string]*db.Sentinel, len(eligibleByRegion))
		for region, sentinel := range eligibleByRegion {
			if s.resilienceTracker.Allow(region, now) {
				filtered[region] = sentinel
			}
		}

		if len(filtered) == 0 {
			s.logger.Warn("all eligible regions excluded by circuit breaker, failing open",
				"eligibleCount", len(eligibleByRegion),
				"deploymentID", route.DeploymentID,
			)
		} else {
			eligibleByRegion = filtered
		}
	}

	if localGw, ok := eligibleByRegion[s.region]; ok {
		decision.LocalSentinel = localGw
		span.SetAttributes(
			attribute.Bool("local_sentinel", true),
			attribute.String("selected_region", s.region),
		)
		return decision, nil
	}

	nearestRegion := s.findNearestRegion(eligibleByRegion)
	if nearestRegion != "" {
		decision.NearestNLBRegion = nearestRegion
		span.SetAttributes(
			attribute.Bool("local_sentinel", false),
			attribute.String("selected_region", nearestRegion),
		)
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

package router

import (
	"context"

	internalCaches "github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// regionProximity maps AWS regions to their closest regions in order of proximity.
var regionProximity = map[string][]string{
	// US East
	"us-east-1": {"us-east-2", "us-west-2", "us-west-1", "ca-central-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},
	"us-east-2": {"us-east-1", "us-west-2", "us-west-1", "ca-central-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},

	// US West
	"us-west-1": {"us-west-2", "us-east-2", "us-east-1", "ca-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "eu-west-1", "eu-central-1"},
	"us-west-2": {"us-west-1", "us-east-2", "us-east-1", "ca-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "eu-west-1", "eu-central-1"},

	// Canada
	"ca-central-1": {"us-east-2", "us-east-1", "us-west-2", "us-west-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},

	// Europe
	"eu-west-1":    {"eu-west-2", "eu-central-1", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-west-2":    {"eu-west-1", "eu-central-1", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-central-1": {"eu-west-1", "eu-west-2", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-north-1":   {"eu-central-1", "eu-west-1", "eu-west-2", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},

	// Asia Pacific
	"ap-south-1":     {"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "eu-west-1", "eu-central-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2"},
	"ap-northeast-1": {"ap-southeast-1", "ap-southeast-2", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
	"ap-southeast-1": {"ap-southeast-2", "ap-northeast-1", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
	"ap-southeast-2": {"ap-southeast-1", "ap-northeast-1", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
}

type service struct {
	logger                     logging.Logger
	region                     string
	db                         db.Database
	ingressRouteCache          cache.Cache[string, db.IngressRoute]
	gatewaysByEnvironmentCache cache.Cache[string, []db.Gateway]
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	return &service{
		logger:                     cfg.Logger,
		region:                     cfg.Region,
		db:                         cfg.DB,
		ingressRouteCache:          cfg.IngressRouteCache,
		gatewaysByEnvironmentCache: cfg.GatewaysByEnvironment,
	}, nil
}

func (s *service) LookupByHostname(ctx context.Context, hostname string) (*db.IngressRoute, []db.Gateway, error) {
	route, hit, err := s.ingressRouteCache.SWR(ctx, hostname, func(ctx context.Context) (db.IngressRoute, error) {
		return db.Query.FindIngressRouteByHostname(ctx, s.db.RO(), hostname)
	}, internalCaches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading ingress route"),
			fault.Public("Failed to load route configuration"),
		)
	}

	if db.IsNotFound(err) || hit == cache.Null {
		return nil, nil, fault.New("no ingress route for hostname: "+hostname,
			fault.Code(codes.Ingress.Routing.ConfigNotFound.URN()),
			fault.Public("Domain not configured"),
		)
	}

	gateways, _, err := s.gatewaysByEnvironmentCache.SWR(ctx, route.EnvironmentID, func(ctx context.Context) ([]db.Gateway, error) {
		return db.Query.FindGatewaysByEnvironmentID(ctx, s.db.RO(), route.EnvironmentID)
	}, internalCaches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return nil, nil, fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading gateways"),
			fault.Public("Failed to load gateway configuration"),
		)
	}

	return &route, gateways, nil
}

func (s *service) SelectGateway(route *db.IngressRoute, gateways []db.Gateway) (*RouteDecision, error) {
	decision := &RouteDecision{
		DeploymentID:     route.DeploymentID,
		LocalGateway:     nil,
		NearestNLBRegion: "",
	}

	healthyByRegion := make(map[string]*db.Gateway)
	for i := range gateways {
		gw := &gateways[i]
		if !gw.Health.Valid ||
			gw.Health.GatewaysHealth != db.GatewaysHealthHealthy {
			continue
		}

		healthyByRegion[gw.Region] = gw
	}

	if len(healthyByRegion) == 0 {
		return nil, fault.New("no healthy gateways",
			fault.Code(codes.Ingress.Routing.NoRunningInstances.URN()),
			fault.Internal("no healthy gateways for environment"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	if localGw, ok := healthyByRegion[s.region]; ok {
		decision.LocalGateway = localGw
		return decision, nil
	}

	nearestRegion := s.findNearestRegion(healthyByRegion)
	if nearestRegion != "" {
		decision.NearestNLBRegion = nearestRegion
	}

	return decision, nil
}

func (s *service) findNearestRegion(healthyByRegion map[string]*db.Gateway) string {
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

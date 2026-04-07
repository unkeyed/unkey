package router

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type service struct {
	platform                    string
	region                      string
	regionPlatform              string
	portalAddr                  string
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
		portalAddr:                  cfg.PortalAddr,
		db:                          cfg.DB,
		frontlineRouteCache:         cfg.FrontlineRouteCache,
		sentinelsByEnvironmentCache: cfg.SentinelsByEnvironment,
		instancesByDeploymentCache:  cfg.InstancesByDeployment,
	}, nil
}

func (s *service) Route(ctx context.Context, hostname string) (RouteDecision, error) {
	start := time.Now()

	route, err := s.findRoute(ctx, hostname)
	if err != nil {
		return RouteDecision{}, err
	}

	// Portal routes skip sentinel entirely — forward directly to the portal service.
	if route.RouteType == db.FrontlineRoutesRouteTypePortal {
		return s.routePortal(route)
	}

	sentinels, err := s.lookupSentinels(ctx, route)
	if err != nil {
		return RouteDecision{}, err
	}

	instances, err := s.getInstances(ctx, route.DeploymentID.String)
	if err != nil {
		routingErrorsTotal.WithLabelValues("instance_load_failed").Inc()
		return RouteDecision{}, err
	}

	decision, err := s.selectSentinel(route, sentinels, instances)

	routingDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		return RouteDecision{}, err
	}

	if decision.Destination == DestinationLocalSentinel {
		routingDecisionsTotal.WithLabelValues("local_sentinel", s.regionPlatform).Inc()
	} else {
		routingDecisionsTotal.WithLabelValues("remote_region", decision.Address).Inc()
	}

	return decision, nil
}

// routePortal builds a RouteDecision for portal routes. Portal routes bypass
// sentinel and forward directly to the portal service.
func (s *service) routePortal(route db.FindFrontlineRouteByFQDNRow) (RouteDecision, error) {
	if s.portalAddr == "" {
		return RouteDecision{}, fault.New("portal service not configured",
			fault.Code(codes.Frontline.Proxy.ServiceUnavailable.URN()),
			fault.Internal("portal_addr is empty, cannot route portal request"),
			fault.Public("Portal service temporarily unavailable"),
		)
	}

	return RouteDecision{
		DeploymentID: "",
		Destination:  DestinationPortal,
		Address:      s.portalAddr,
		PathPrefix:   route.PathPrefix.String,
	}, nil
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.findRoute(ctx, hostname)
	return err
}

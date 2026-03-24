package router

import (
	"context"

	internalCaches "github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

func (s *service) findRoute(ctx context.Context, hostname string) (db.FindFrontlineRouteByFQDNRow, error) {
	route, routeHit, err := s.frontlineRouteCache.SWR(ctx, hostname, func(ctx context.Context) (db.FindFrontlineRouteByFQDNRow, error) {
		return s.db.FindFrontlineRouteByFQDN(ctx, hostname)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		routingErrorsTotal.WithLabelValues("config_load_failed").Inc()
		return db.FindFrontlineRouteByFQDNRow{}, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading frontline route"),
			fault.Public("Failed to load route configuration"),
		)
	}

	if mysql.IsNotFound(err) || routeHit == cache.Null {
		routingErrorsTotal.WithLabelValues("config_not_found").Inc()
		return db.FindFrontlineRouteByFQDNRow{}, fault.New("no frontline route for hostname: "+hostname,
			fault.Code(codes.Frontline.Routing.ConfigNotFound.URN()),
			fault.Public("Domain not configured"),
		)
	}

	return route, nil
}

func (s *service) lookupByHostname(ctx context.Context, hostname string) (db.FindFrontlineRouteByFQDNRow, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
	route, err := s.findRoute(ctx, hostname)
	if err != nil {
		return db.FindFrontlineRouteByFQDNRow{}, nil, err
	}

	sentinels, _, err := s.sentinelsByEnvironmentCache.SWR(ctx, route.EnvironmentID, func(ctx context.Context) ([]db.FindHealthyRoutableSentinelsByEnvironmentIDRow, error) {
		return s.db.FindHealthyRoutableSentinelsByEnvironmentID(ctx, route.EnvironmentID)
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		routingErrorsTotal.WithLabelValues("sentinel_load_failed").Inc()
		return db.FindFrontlineRouteByFQDNRow{}, nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading sentinels"),
			fault.Public("Failed to load sentinel configuration"),
		)
	}

	if sentinels == nil {
		sentinels = []db.FindHealthyRoutableSentinelsByEnvironmentIDRow{}
	}

	return route, sentinels, nil
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

	if instances == nil {
		instances = []db.Instance{}
	}

	return instances, nil
}

package router

import (
	"context"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	internalCaches "github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
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

func (s *service) getInstances(ctx context.Context, deploymentID string) ([]db.FindInstancesByDeploymentIDRow, error) {
	instances, _, err := s.instancesByDeploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.FindInstancesByDeploymentIDRow, error) {
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
		instances = []db.FindInstancesByDeploymentIDRow{}
	}

	return instances, nil
}

// getPolicies parses the sentinel_config bytes carried on the route into a
// slice of policies. The parse result is cached by deployment_id so cluster-
// wide invalidation flushes both the route and policy caches together.
func (s *service) getPolicies(ctx context.Context, route db.FindFrontlineRouteByFQDNRow) ([]*frontlinev1.Policy, error) {
	policies, hit, err := s.policyCache.SWR(ctx, route.DeploymentID, func(ctx context.Context) ([]*frontlinev1.Policy, error) {
		return policies.ParseMiddleware(route.SentinelConfig)
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})

	if err != nil {
		return nil, err
	}

	if hit == cache.Null {
		return nil, nil
	}

	return policies, nil
}

package router

import (
	"context"
	"database/sql"

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
//
// OpenAPI policies that carry no inline spec are hydrated from the scraped
// spec in openapi_specs, keyed by deployment_id.
func (s *service) getPolicies(ctx context.Context, route db.FindFrontlineRouteByFQDNRow) ([]*frontlinev1.Policy, error) {
	pols, hit, err := s.policyCache.SWR(ctx, route.DeploymentID, func(ctx context.Context) ([]*frontlinev1.Policy, error) {
		parsed, parseErr := policies.ParseMiddleware(route.SentinelConfig)
		if parseErr != nil {
			return nil, parseErr
		}

		if err := s.hydrateOpenapiSpecs(ctx, route.DeploymentID, parsed); err != nil {
			return nil, err
		}

		return parsed, nil
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

	return pols, nil
}

// hydrateOpenapiSpecs loads the spec from openapi_specs (keyed by deployment_id)
// and sets it on every openapi policy. Specs always come from the DB.
func (s *service) hydrateOpenapiSpecs(ctx context.Context, deploymentID string, pols []*frontlinev1.Policy) error {
	var targets []*frontlinev1.OpenApiRequestValidation
	for _, p := range pols {
		if oa, ok := p.GetConfig().(*frontlinev1.Policy_Openapi); ok {
			targets = append(targets, oa.Openapi)
		}
	}
	if len(targets) == 0 {
		return nil
	}

	spec, err := s.db.FindOpenApiSpecByDeploymentID(ctx, sql.NullString{String: deploymentID, Valid: true})
	if err != nil {
		if mysql.IsNotFound(err) {
			return nil
		}
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading openapi spec for deployment "+deploymentID),
		)
	}

	for _, t := range targets {
		t.SpecYaml = spec
	}

	return nil
}

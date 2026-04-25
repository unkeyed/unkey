package router

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/services/edgeredirect"
)

type service struct {
	platform                    string
	region                      string
	regionPlatform              string
	db                          db.Querier
	frontlineRouteCache         cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	sentinelsByEnvironmentCache cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
	instancesByDeploymentCache  cache.Cache[string, []db.FindInstancesByDeploymentIDRow]
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

func (s *service) Route(ctx context.Context, hostname string) (RouteDecision, error) {
	start := time.Now()

	route, sentinels, err := s.lookupByHostname(ctx, hostname)
	if err != nil {
		return RouteDecision{}, err
	}

	instances, err := s.getInstances(ctx, route.DeploymentID)
	if err != nil {
		routingErrorsTotal.WithLabelValues("instance_load_failed").Inc()
		return RouteDecision{}, err
	}

	decision, err := s.selectSentinel(route, sentinels, instances)

	routingDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		return RouteDecision{}, err
	}

	// Parse the edge-redirect rules attached to the route. A bad blob is
	// non-fatal: we log and serve the request without redirects so a
	// malformed config can't take a customer offline. The parse itself is
	// microseconds and only matters once per request.
	if rules, parseErr := edgeredirect.ParseConfig(route.EdgeRedirectConfig); parseErr != nil {
		slog.Warn("edge-redirect config parse failed; serving with no redirects",
			"hostname", hostname,
			"error", parseErr,
		)
	} else {
		decision.Redirects = rules
	}

	if decision.Destination == DestinationLocalSentinel {
		routingDecisionsTotal.WithLabelValues("local_sentinel", s.regionPlatform).Inc()
	} else {
		routingDecisionsTotal.WithLabelValues("remote_region", decision.Address).Inc()
	}

	return decision, nil
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.findRoute(ctx, hostname)
	return err
}

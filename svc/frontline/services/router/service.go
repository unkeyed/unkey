package router

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type service struct {
	platform                    string
	region                      string
	regionPlatform              string
	db                          db.Querier
	frontlineRouteCache         cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	sentinelsByEnvironmentCache cache.Cache[string, []db.FindHealthyRoutableSentinelsByEnvironmentIDRow]
	instancesByDeploymentCache  cache.Cache[string, []db.FindInstancesByDeploymentIDRow]
	metrics                     *Metrics
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
		metrics:                     cfg.Metrics,
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
		s.metrics.ErrorsTotal.WithLabelValues("instance_load_failed").Inc()
		return RouteDecision{}, err
	}

	decision, err := s.selectSentinel(route, sentinels, instances)

	s.metrics.Duration.Observe(time.Since(start).Seconds())

	if err != nil {
		return RouteDecision{}, err
	}

	if decision.Destination == DestinationLocalSentinel {
		s.metrics.DecisionsTotal.WithLabelValues("local_sentinel", s.regionPlatform).Inc()
	} else {
		s.metrics.DecisionsTotal.WithLabelValues("remote_region", decision.Address).Inc()
	}

	return decision, nil
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.findRoute(ctx, hostname)
	return err
}

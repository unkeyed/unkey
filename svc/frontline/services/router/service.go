package router

import (
	"context"
	"fmt"

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
	_, err := s.findRoute(ctx, hostname)
	return err
}

package router

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
)

type service struct {
	platform                   string
	region                     string
	regionPlatform             string
	db                         db.Querier
	frontlineRouteCache        cache.Cache[string, db.FindFrontlineRouteByFQDNRow]
	instancesByDeploymentCache cache.Cache[string, []db.FindInstancesByDeploymentIDRow]
	policyCache                cache.Cache[string, []*frontlinev1.Policy]

	// localDecisions is routingDecisionsTotal pre-bound to the local
	// decision labels, which are constant for the process. Binding once
	// avoids the label hashing WithLabelValues does on every request.
	// Remote decisions stay dynamic — their target region varies.
	localDecisions prometheus.Counter
}

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	regionPlatform := cfg.Region + "." + cfg.Platform
	return &service{
		platform:                   cfg.Platform,
		region:                     cfg.Region,
		regionPlatform:             regionPlatform,
		db:                         cfg.DB,
		frontlineRouteCache:        cfg.FrontlineRouteCache,
		instancesByDeploymentCache: cfg.InstancesByDeployment,
		policyCache:                cfg.PolicyCache,
		localDecisions:             routingDecisionsTotal.WithLabelValues(decisionLocal, regionPlatform),
	}, nil
}

func (s *service) Route(ctx context.Context, hostname string) (RouteDecision, error) {
	start := time.Now()
	defer func() { routingDecisionSeconds.Observe(time.Since(start).Seconds()) }()

	route, err := s.findRoute(ctx, hostname)
	if err != nil {
		return RouteDecision{}, err
	}

	instances, err := s.getInstances(ctx, route.DeploymentID)
	if err != nil {
		return RouteDecision{}, err
	}

	policies, err := s.getPolicies(ctx, route)
	if err != nil {
		return RouteDecision{}, err
	}

	decision, err := s.selectDestination(route, instances, policies)
	if err != nil {
		return RouteDecision{}, err
	}

	if decision.Destination == DestinationLocalInstance {
		s.localDecisions.Inc()
	} else {
		routingDecisionsTotal.WithLabelValues(decisionRemote, decision.RemoteRegionAddress).Inc()
	}

	return decision, nil
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.findRoute(ctx, hostname)
	return err
}

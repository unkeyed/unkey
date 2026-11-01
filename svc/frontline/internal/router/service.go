package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
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
	clock                      clock.Clock

	// reloadsWithoutRunning holds, per deployment whose cached instances
	// have none running, when they were last reloaded from the database.
	reloadsWithoutRunning sync.Map
}

// reloadIntervalWithoutRunning is how often Route reloads cached instances
// that have none running. Without it, a deployment whose replicas all
// restarted returns 503 for the rest of the cache's fresh window after its
// pods are ready again.
const reloadIntervalWithoutRunning = time.Second

var _ Service = (*service)(nil)

func New(cfg Config) (*service, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}
	//nolint:exhaustruct
	return &service{
		clock:                      clk,
		platform:                   cfg.Platform,
		region:                     cfg.Region,
		regionPlatform:             fmt.Sprintf("%s.%s", cfg.Region, cfg.Platform),
		db:                         cfg.DB,
		frontlineRouteCache:        cfg.FrontlineRouteCache,
		instancesByDeploymentCache: cfg.InstancesByDeployment,
		policyCache:                cfg.PolicyCache,
	}, nil
}

func (s *service) Route(ctx context.Context, hostname string, excludedRegions ...string) (RouteDecision, error) {
	start := time.Now()
	defer func() { routingDecisionSeconds.Observe(time.Since(start).Seconds()) }()

	route, err := s.findRoute(ctx, hostname)
	if err != nil {
		return RouteDecision{}, err
	}
	decision := RouteDecision{
		DeploymentID:        route.DeploymentID,
		EnvironmentID:       route.EnvironmentID,
		WorkspaceID:         route.WorkspaceID,
		ProjectID:           route.ProjectID,
		AppID:               route.AppID,
		UpstreamProtocol:    route.UpstreamProtocol,
		Destination:         "",
		LocalInstances:      nil,
		RemoteRegionAddress: "",
		Policies:            nil,
	}

	// A workspace paused for reaching its Compute spend limit is a billing gate:
	// refuse traffic with a 402 up front, before the desired-state check and
	// before instance selection. spend_suspended is the authoritative "paused"
	// signal, so this fires even while the stop is still propagating (the
	// deployment's desired_state has not flipped, or an instance is still
	// draining). Resumes when the budget is raised or removed.
	if route.SpendSuspended.Valid && route.SpendSuspended.Bool {
		return decision, fault.New("workspace spend limit reached",
			fault.Code(codes.Frontline.Routing.SpendLimitReached.URN()),
			fault.Internal(fmt.Sprintf("workspace spend-cap suspended; deployment %s", route.DeploymentID)),
			fault.Public("Compute for this workspace is paused because it reached its spend limit."),
		)
	}

	// A stopped or archived deployment was torn down on purpose, so it is
	// intentionally offline rather than transiently between instances like the
	// no_running_instances seen mid-deploy or mid-scale.
	if route.DesiredState != db.DeploymentsDesiredStateRunning {
		return decision, fault.New("deployment offline",
			fault.Code(codes.Frontline.Routing.DeploymentOffline.URN()),
			fault.Internal(fmt.Sprintf("deployment %s desired_state=%s", route.DeploymentID, route.DesiredState)),
			fault.Public("This deployment is offline."),
		)
	}

	instances, err := s.getInstances(ctx, route.DeploymentID)
	if err != nil {
		return decision, err
	}
	if s.shouldReloadWithoutRunning(route.DeploymentID, instances) {
		s.instancesByDeploymentCache.Remove(ctx, route.DeploymentID)
		instances, err = s.getInstances(ctx, route.DeploymentID)
		if err != nil {
			return RouteDecision{}, err
		}
	}

	policies, err := s.getPolicies(ctx, route)
	if err != nil {
		return decision, err
	}

	decision, err = s.selectDestination(decision, instances, policies, excludedRegions)
	if err != nil {
		return decision, err
	}

	if decision.Destination == DestinationLocalInstance {
		routingDecisionsTotal.WithLabelValues(decisionLocal, s.regionPlatform).Inc()
	} else {
		routingDecisionsTotal.WithLabelValues(decisionRemote, decision.RemoteRegionAddress).Inc()
	}

	return decision, nil
}

func (s *service) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.findRoute(ctx, hostname)
	return err
}

func (s *service) InvalidateInstances(ctx context.Context, deploymentID string) {
	s.instancesByDeploymentCache.Remove(ctx, deploymentID)
}

// shouldReloadWithoutRunning reports whether the cached instances of a
// deployment have none running and were last reloaded at least
// reloadIntervalWithoutRunning ago. The first sighting only starts the
// interval, because those instances may have just been loaded. Only one
// concurrent caller per deployment gets true.
func (s *service) shouldReloadWithoutRunning(deploymentID string, instances []db.FindInstancesByDeploymentIDRow) bool {
	for _, inst := range instances {
		if inst.Status == db.InstancesStatusRunning {
			s.reloadsWithoutRunning.Delete(deploymentID)
			return false
		}
	}

	now := s.clock.Now()
	last, loaded := s.reloadsWithoutRunning.LoadOrStore(deploymentID, now)
	if !loaded || now.Sub(last.(time.Time)) < reloadIntervalWithoutRunning {
		return false
	}
	return s.reloadsWithoutRunning.CompareAndSwap(deploymentID, last, now)
}

package router

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/prometheus/timer"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/sentinel/engine"
)

var _ Service = (*service)(nil)

type service struct {
	db            db.Database
	clock         clock.Clock
	environmentID string
	platform      string
	region        string

	// deploymentID -> deployment
	deploymentCache cache.Cache[string, db.Deployment]
	// deploymentID -> instances
	instancesCache cache.Cache[string, []db.Instance]
	// deploymentID -> parsed sentinel policies (avoids proto unmarshal on every request)
	policyCache cache.Cache[string, []*sentinelv1.Policy]

	// dispatcher handles routing of invalidation events to all caches in this service.
	dispatcher *clustering.InvalidationDispatcher
}

// Close shuts down the service and cleans up resources.
func (s *service) Close() error {
	if s.dispatcher != nil {
		return s.dispatcher.Close()
	}

	return nil
}

// clusterOpts bundles the dispatcher and key converter functions needed for
// distributed cache invalidation.
type clusterOpts[K comparable] struct {
	dispatcher  *clustering.InvalidationDispatcher
	broadcaster clustering.Broadcaster
	nodeID      string
	keyToString func(K) string
	stringToKey func(string) (K, error)
}

// createCache creates a cache instance with optional clustering support.
func createCache[K comparable, V any](
	cacheConfig cache.Config[K, V],
	opts *clusterOpts[K],
) (cache.Cache[K, V], error) {
	localCache, err := cache.New(cacheConfig)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		return localCache, nil
	}

	clusterCache, err := clustering.New(clustering.Config[K, V]{
		LocalCache:  localCache,
		Broadcaster: opts.broadcaster,
		Dispatcher:  opts.dispatcher,
		NodeID:      opts.nodeID,
		KeyToString: opts.keyToString,
		StringToKey: opts.stringToKey,
	})
	if err != nil {
		return nil, err
	}

	return clusterCache, nil
}

func New(cfg Config) (*service, error) {
	nodeID := cfg.NodeID
	if nodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		nodeID = fmt.Sprintf("%s-%s", hostname, uid.New("node"))
	}

	var dispatcher *clustering.InvalidationDispatcher
	var stringKeyOpts *clusterOpts[string]

	if cfg.Broadcaster != nil {
		var err error
		dispatcher, err = clustering.NewInvalidationDispatcher(cfg.Broadcaster)
		if err != nil {
			return nil, err
		}

		stringKeyOpts = &clusterOpts[string]{
			dispatcher:  dispatcher,
			broadcaster: cfg.Broadcaster,
			nodeID:      nodeID,
			keyToString: nil,
			stringToKey: nil,
		}
	}

	deploymentCache, err := createCache(
		cache.Config[string, db.Deployment]{
			Resource: "deployment",
			Clock:    cfg.Clock,
			MaxSize:  1000,
			Fresh:    30 * time.Second,
			Stale:    5 * time.Minute,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, err
	}

	instancesCache, err := createCache(
		cache.Config[string, []db.Instance]{
			Clock:    cfg.Clock,
			Resource: "instance",
			MaxSize:  1000,
			Fresh:    10 * time.Second,
			Stale:    60 * time.Second,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, err
	}

	// Policy cache uses the same TTLs as the deployment cache since policies
	// are derived from the deployment's SentinelConfig.
	policyCache, err := createCache(
		cache.Config[string, []*sentinelv1.Policy]{
			Clock:    cfg.Clock,
			Resource: "policy",
			MaxSize:  1000,
			Fresh:    30 * time.Second,
			Stale:    5 * time.Minute,
		},
		stringKeyOpts,
	)
	if err != nil {
		return nil, err
	}

	s := &service{
		db:              cfg.DB,
		clock:           cfg.Clock,
		environmentID:   cfg.EnvironmentID,
		platform:        cfg.Platform,
		region:          cfg.Region,
		deploymentCache: deploymentCache,
		instancesCache:  instancesCache,
		policyCache:     policyCache,
		dispatcher:      dispatcher,
	}

	go s.prewarm(context.Background())
	return s, nil
}

func (s *service) prewarm(ctx context.Context) {
	logger.Info("prewarming cache")

	deployments, err := db.Query.ListDeploymentsByEnvironmentIdAndStatus(ctx, s.db.RO(), db.ListDeploymentsByEnvironmentIdAndStatusParams{
		EnvironmentID: s.environmentID,
		Status:        db.DeploymentsStatusReady,
		CreatedBefore: time.Now().UnixMilli(),
		UpdatedBefore: sql.NullInt64{Valid: false, Int64: 0},
	})
	if err != nil {
		logger.Error("unable to prewarm deployment cache", "error", err.Error())
		return
	}

	region, err := db.Query.FindRegionByPlatformAndName(ctx, s.db.RO(), db.FindRegionByPlatformAndNameParams{
		Platform: s.platform,
		Name:     s.region,
	})
	if err != nil {
		logger.Error("unable to find region for prewarming instance cache", "platform", s.platform, "region", s.region, "error", err.Error())
		return
	}

	for _, d := range deployments {
		instances, err := db.Query.FindInstancesByDeploymentIdAndRegionID(ctx, s.db.RO(), db.FindInstancesByDeploymentIdAndRegionIDParams{
			DeploymentID: d.ID,
			RegionID:     region.ID,
		})
		if err != nil {
			logger.Error("unable to find instances for deployment", "deployment_id", d.ID, "error", err.Error())
			continue
		}

		logger.Info("precaching deployment", "deployment_id", d.ID)
		s.deploymentCache.Set(ctx, d.ID, d)
		s.instancesCache.Set(ctx, d.ID, instances)

		policies, parseErr := engine.ParseMiddleware(d.SentinelConfig)
		if parseErr != nil {
			logger.Error("unable to parse sentinel config for deployment", "deployment_id", d.ID, "error", parseErr.Error())
		} else if policies != nil {
			s.policyCache.Set(ctx, d.ID, policies)
		}
	}

	logger.Info("deployment and instance cache are warm")
}

func (s *service) GetDeployment(ctx context.Context, deploymentID string) (db.Deployment, error) {
	t := timer.New()

	deployment, hit, err := s.deploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) (db.Deployment, error) {
		return db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	}, caches.DefaultFindFirstOp)

	sentinelRoutingDuration.WithLabelValues("get_deployment").Observe(t.Seconds())

	if err != nil && !db.IsNotFound(err) {
		sentinelInstanceSelectionTotal.WithLabelValues("error").Inc()
		return db.Deployment{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get deployment"),
		)
	}

	if hit == cache.Null || db.IsNotFound(err) {
		sentinelInstanceSelectionTotal.WithLabelValues("deployment_not_found").Inc()
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Sentinel.Routing.DeploymentNotFound.URN()),
			fault.Internal("no deployment found for ID or wrong environment"),
			fault.Public("Deployment not found"),
		)
	}

	if deployment.EnvironmentID != s.environmentID {
		logger.Warn("deployment does not belong to this environment",
			"deploymentID", deploymentID,
			"deploymentEnv", deployment.EnvironmentID,
			"sentinelEnv", s.environmentID,
		)

		sentinelInstanceSelectionTotal.WithLabelValues("deployment_not_found").Inc()
		// Return as not found to avoid leaking information about deployments in other environments
		return db.Deployment{}, fault.New("deployment not found",
			fault.Code(codes.Sentinel.Routing.DeploymentNotFound.URN()),
			fault.Internal(fmt.Sprintf("deployment %s belongs to environment %s, but sentinel serves %s", deploymentID, deployment.EnvironmentID, s.environmentID)),
			fault.Public("Deployment not found"),
		)
	}

	return deployment, nil
}

func (s *service) SelectInstance(ctx context.Context, deploymentID string) (db.Instance, error) {
	t := timer.New()

	instances, hit, err := s.instancesCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.Instance, error) {

		region, err := db.Query.FindRegionByPlatformAndName(ctx, s.db.RO(), db.FindRegionByPlatformAndNameParams{
			Platform: s.platform,
			Name:     s.region,
		})
		if err != nil {
			return nil, err
		}

		return db.Query.FindInstancesByDeploymentIdAndRegionID(
			ctx,
			s.db.RO(),
			db.FindInstancesByDeploymentIdAndRegionIDParams{
				DeploymentID: deploymentID,
				RegionID:     region.ID,
			},
		)
	}, caches.DefaultFindFirstOp)

	sentinelRoutingDuration.WithLabelValues("select_instance").Observe(t.Seconds())

	if err != nil {
		sentinelInstanceSelectionTotal.WithLabelValues("error").Inc()
		return db.Instance{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get instances"),
		)
	}

	if hit == cache.Null || len(instances) == 0 {
		sentinelInstanceSelectionTotal.WithLabelValues("no_instances").Inc()
		return db.Instance{}, fault.New("no instances found",
			fault.Code(codes.Sentinel.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no instances for deployment %s in region %s", deploymentID, s.region)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	var runningInstances []db.Instance
	for _, instance := range instances {
		if instance.Status == db.InstancesStatusRunning {
			runningInstances = append(runningInstances, instance)
		}
	}

	if len(runningInstances) == 0 {
		sentinelInstanceSelectionTotal.WithLabelValues("no_running_instances").Inc()
		return db.Instance{}, fault.New("no running instances",
			fault.Code(codes.Sentinel.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s in region %s (found %d total)", deploymentID, s.region, len(instances))),
			fault.Public("Service temporarily unavailable"),
		)
	}

	sentinelInstanceSelectionTotal.WithLabelValues("success").Inc()
	selected := array.Random(runningInstances)
	return selected, nil
}

func (s *service) GetPolicies(ctx context.Context, deployment db.Deployment) ([]*sentinelv1.Policy, error) {
	policies, hit, err := s.policyCache.SWR(ctx, deployment.ID, func(ctx context.Context) ([]*sentinelv1.Policy, error) {
		return engine.ParseMiddleware(deployment.SentinelConfig)
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

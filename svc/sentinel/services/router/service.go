package router

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/sentinel/internal/db"
)

var _ Service = (*service)(nil)

type service struct {
	db            db.Querier
	clock         clock.Clock
	environmentID string
	platform      string
	region        string
	regionID      string

	// deploymentID -> deployment
	deploymentCache cache.Cache[string, db.FindDeploymentByIdRow]
	// deploymentID -> instances
	instancesCache cache.Cache[string, []db.FindInstancesByDeploymentIdAndRegionIDRow]

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
		cache.Config[string, db.FindDeploymentByIdRow]{
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
		cache.Config[string, []db.FindInstancesByDeploymentIdAndRegionIDRow]{
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

	// Temporary fallback until all sentinels receive region_id from krane.
	if cfg.RegionID == "" {
		regionID, err := cfg.DB.FindRegionByPlatformAndName(context.Background(), db.FindRegionByPlatformAndNameParams{
			Platform: cfg.Platform,
			Name:     cfg.Region,
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
				fault.Internal(fmt.Sprintf("failed to find region ID for platform %s and region %s", cfg.Platform, cfg.Region)),
			)
		}
		cfg.RegionID = regionID
	}

	s := &service{
		db:              cfg.DB,
		clock:           cfg.Clock,
		environmentID:   cfg.EnvironmentID,
		platform:        cfg.Platform,
		region:          cfg.Region,
		regionID:        cfg.RegionID,
		deploymentCache: deploymentCache,
		instancesCache:  instancesCache,
		dispatcher:      dispatcher,
	}

	go s.prewarm(context.Background())
	return s, nil
}

func (s *service) prewarm(ctx context.Context) {
	logger.Info("prewarming cache")

	deployments, err := s.db.ListDeploymentsByEnvironmentIdAndStatus(ctx, db.ListDeploymentsByEnvironmentIdAndStatusParams{
		EnvironmentID: s.environmentID,
		Status:        db.DeploymentsStatusReady,
		CreatedBefore: time.Now().UnixMilli(),
		UpdatedBefore: sql.NullInt64{Valid: false, Int64: 0},
	})
	if err != nil {
		logger.Error("unable to prewarm deployment cache", "error", err.Error())
		return
	}

	for _, d := range deployments {
		instanceRows, err := s.db.FindInstancesByDeploymentIdAndRegionID(ctx, db.FindInstancesByDeploymentIdAndRegionIDParams{
			DeploymentID: d.ID,
			RegionID:     s.regionID,
			Status:       db.InstancesStatusRunning,
		})
		if err != nil {
			logger.Error("unable to find instances for deployment", "deployment_id", d.ID, "error", err.Error())
			continue
		}

		logger.Info("precaching deployment", "deployment_id", d.ID)
		s.deploymentCache.Set(ctx, d.ID, db.FindDeploymentByIdRow{
			ID:             d.ID,
			WorkspaceID:    d.WorkspaceID,
			ProjectID:      d.ProjectID,
			EnvironmentID:  d.EnvironmentID,
			SentinelConfig: d.SentinelConfig,
		})
		s.instancesCache.Set(ctx, d.ID, instanceRows)
	}

	logger.Info("deployment and instance cache are warm")
}

func (s *service) GetDeployment(ctx context.Context, deploymentID string) (db.FindDeploymentByIdRow, error) {
	deployment, hit, err := s.deploymentCache.SWR(ctx, deploymentID, func(ctx context.Context) (db.FindDeploymentByIdRow, error) {
		return s.db.FindDeploymentById(ctx, deploymentID)
	}, caches.DefaultFindFirstOp)

	if err != nil && !mysql.IsNotFound(err) {
		return db.FindDeploymentByIdRow{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get deployment"),
		)
	}

	if hit == cache.Null || mysql.IsNotFound(err) {
		return db.FindDeploymentByIdRow{}, fault.New("deployment not found",
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

		// Return as not found to avoid leaking information about deployments in other environments
		return db.FindDeploymentByIdRow{}, fault.New("deployment not found",
			fault.Code(codes.Sentinel.Routing.DeploymentNotFound.URN()),
			fault.Internal(fmt.Sprintf("deployment %s belongs to environment %s, but sentinel serves %s", deploymentID, deployment.EnvironmentID, s.environmentID)),
			fault.Public("Deployment not found"),
		)
	}

	return deployment, nil
}

func (s *service) SelectInstance(ctx context.Context, deploymentID string) (db.FindInstancesByDeploymentIdAndRegionIDRow, error) {
	instances, hit, err := s.instancesCache.SWR(ctx, deploymentID, func(ctx context.Context) ([]db.FindInstancesByDeploymentIdAndRegionIDRow, error) {

		instanceRows, findErr := s.db.FindInstancesByDeploymentIdAndRegionID(
			ctx,
			db.FindInstancesByDeploymentIdAndRegionIDParams{
				DeploymentID: deploymentID,
				RegionID:     s.regionID,
				Status:       db.InstancesStatusRunning,
			},
		)
		if findErr != nil {
			return nil, findErr
		}

		return instanceRows, nil
	}, caches.DefaultFindFirstOp)

	if err != nil {
		return db.FindInstancesByDeploymentIdAndRegionIDRow{}, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to get instances"),
		)
	}

	if hit == cache.Null || len(instances) == 0 {
		return db.FindInstancesByDeploymentIdAndRegionIDRow{}, fault.New("no running instances",
			fault.Code(codes.Sentinel.Routing.NoRunningInstances.URN()),
			fault.Internal(fmt.Sprintf("no running instances for deployment %s in region %s", deploymentID, s.region)),
			fault.Public("Service temporarily unavailable"),
		)
	}

	selected := array.Random(instances)
	return selected, nil
}

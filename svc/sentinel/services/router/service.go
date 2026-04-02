package router

import (
	"context"
	"fmt"
	"os"
	"time"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/sentinel/services/balancer"
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

	// balancer selects instances for load balancing.
	balancer balancer.Balancer
}

// Close shuts down the service and cleans up resources.
func (s *service) Close() error {
	if s.dispatcher != nil {
		return s.dispatcher.Close()
	}

	return nil
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
		balancer:        balancer.NewP2CBalancer(),
	}

	go s.prewarm(context.Background())
	return s, nil
}

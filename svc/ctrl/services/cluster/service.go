package cluster

import (
	"time"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// notifiedReadyTTL is how long an entry in notifiedReady is kept before
// being eligible for cleanup. Five minutes is comfortably longer than any
// deployment's notify-ready window — once a deployment transitions to ready
// or terminal there's no value in remembering its entry.
const notifiedReadyTTL = 5 * time.Minute

// Service implements [ctrlv1connect.ClusterServiceHandler] to synchronize desired state
// between the control plane and krane agents. It provides streaming RPCs for watching
// deployment and sentinel changes, point queries for fetching individual resource states,
// and status reporting endpoints for agents to report observed state back to the control plane.
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db      db.Database
	restate *ingress.Client
	bearer  string
	// notifiedReady dedups Restate NotifyInstancesReady calls so we don't
	// fire on every krane status report once the threshold is met. Keys
	// are "deployment:<id>". The sentinel path uses the
	// deploy_status=progressing gate + DB flip as its idempotency
	// mechanism instead (see maybeNotifySentinelReady).
	notifiedReady *expiringSet[string]
	// topologyCache caches FindDeploymentTopologyMinReplicas lookups
	// keyed by deployment_id. Topology is written once at deploy time,
	// then read on every instance status report, so caching removes an
	// RO hit from the notify path. Empty results are not cached (see
	// findTopologyMinReplicas) because a missed race against the
	// topology write must be retried, not sealed in.
	topologyCache cache.Cache[string, []db.FindDeploymentTopologyMinReplicasRow]
}

// Config holds the configuration for creating a new cluster [Service].
type Config struct {
	// Database provides read and write access for querying and updating resource state.
	Database db.Database

	// Restate is the ingress client used to trigger NotifyReady on sentinel virtual objects.
	Restate *ingress.Client

	// Bearer is the authentication token that agents must provide in the Authorization header.
	Bearer string

	// TopologyCache backs FindDeploymentTopologyMinReplicas lookups on
	// the notify-ready path. Required.
	TopologyCache cache.Cache[string, []db.FindDeploymentTopologyMinReplicasRow]
}

// New creates a new cluster [Service] with the given configuration. The returned service
// is ready to be registered with a Connect server. A background sweeper is
// started that periodically drops stale entries from notifiedReady so the
// set doesn't grow unbounded.
func New(cfg Config) *Service {
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		bearer:                             cfg.Bearer,
		notifiedReady:                      newExpiringSet[string](notifiedReadyTTL),
		topologyCache:                      cfg.TopologyCache,
	}
	repeat.Every(notifiedReadyTTL, func() {
		if dropped := s.notifiedReady.Sweep(); dropped > 0 {
			logger.Info("swept stale notifiedReady entries", "dropped", dropped)
		}
	})
	return s
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)

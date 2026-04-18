package cluster

import (
	"fmt"
	"time"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
)

// notifiedReadyTTL is how long an entry in notifiedReady is kept before
// being eligible for cleanup. Five minutes is comfortably longer than any
// deployment's notify-ready window — once a deployment transitions to ready
// or terminal there's no value in remembering its entry.
const notifiedReadyTTL = 5 * time.Minute

// Region lookups are on the hot path of every region-scoped RPC but the
// underlying row is effectively immutable (regions.id never changes once a
// platform/name pair exists), so we cache aggressively. Fresh=5m keeps the
// cache hot for steady-state traffic; Stale=15m lets us tolerate a brief DB
// hiccup without synchronous re-fetch storms.
const (
	regionCacheFresh   = 5 * time.Minute
	regionCacheStale   = 15 * time.Minute
	regionCacheMaxSize = 256
)

// regionCacheKey composes platform and region name into a comparable cache
// key so we don't need a string-encoding helper.
type regionCacheKey struct {
	platform string
	name     string
}

// Service implements [ctrlv1connect.ClusterServiceHandler] to synchronize desired state
// between the control plane and krane agents. It provides streaming RPCs for watching
// deployment and sentinel changes, point queries for fetching individual resource states,
// and status reporting endpoints for agents to report observed state back to the control plane.
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db      db.Database
	restate *ingress.Client
	bearer  string
	// notifiedReady dedups Restate NotifyReady / NotifyInstancesReady calls
	// so we don't fire on every krane status report once the threshold is
	// met. Keys are "deployment:<id>" or "sentinel:<id>".
	notifiedReady *expiringSet[string]
	// regionCache memoizes (platform, name) → [db.Region] lookups via SWR so
	// region-scoped RPCs don't hit the DB on every request.
	regionCache cache.Cache[regionCacheKey, db.Region]
}

// Config holds the configuration for creating a new cluster [Service].
type Config struct {
	// Database provides read and write access for querying and updating resource state.
	Database db.Database

	// Restate is the ingress client used to trigger NotifyReady on sentinel virtual objects.
	Restate *ingress.Client

	// Bearer is the authentication token that agents must provide in the Authorization header.
	Bearer string

	// Clock backs the region cache's freshness accounting. When nil, a
	// real-time clock is used.
	Clock clock.Clock
}

// New creates a new cluster [Service] with the given configuration. The returned service
// is ready to be registered with a Connect server. A background sweeper is
// started that periodically drops stale entries from notifiedReady so the
// set doesn't grow unbounded.
func New(cfg Config) (*Service, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}
	regionCache, err := cache.New(cache.Config[regionCacheKey, db.Region]{
		Fresh:    regionCacheFresh,
		Stale:    regionCacheStale,
		MaxSize:  regionCacheMaxSize,
		Resource: "ctrl_regions",
		Clock:    clk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create region cache: %w", err)
	}
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		bearer:                             cfg.Bearer,
		notifiedReady:                      newExpiringSet[string](notifiedReadyTTL),
		regionCache:                        regionCache,
	}
	repeat.Every(notifiedReadyTTL, func() {
		if dropped := s.notifiedReady.Sweep(); dropped > 0 {
			logger.Info("swept stale notifiedReady entries", "dropped", dropped)
		}
	})
	return s, nil
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)

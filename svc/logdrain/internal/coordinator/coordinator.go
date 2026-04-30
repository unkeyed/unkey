// Package coordinator owns the v1 logdrain control loop: load enabled
// drains from MySQL, filter by shard ownership, group by (workspace,
// project, environment, source), pull a windowed batch from ClickHouse,
// fan out to per-drain sinks, and advance the group cursor under
// optimistic locking once every drain in the group has acknowledged the
// batch.
//
// Cursor semantics, dedup, and the read-amplification design are
// documented in docs/engineering/architecture/services/logdrain. This
// package is the reference implementation of that design.
package coordinator

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
)

// Config carries the policy knobs the coordinator reads from svc/logdrain.
type Config struct {
	PollInterval          time.Duration
	BatchWindow           time.Duration
	MaxBatchRecords       int
	PauseAfterFailures    int
	MaxGroupsPerShard     int
	MaxDrainsPerWorkspace int
	ShardCount            int
	ShardIndex            int
}

// drainListCacheKey is the only key used in drainListCache. The cache is
// effectively a single-slot SWR on the global ListEnabledLogDrains
// query; cache.Cache is keyed on a generic K, so we pass an empty string
// (every tick reads the same slot).
const drainListCacheKey = ""

// Coordinator wires the loaded drains, the ClickHouse client, the sink
// factory, and the MySQL queries together. v1 runs a single instance per
// region; sharding is wired via Config so a future multi-replica rollout
// only needs the helm chart to set ShardCount/ShardIndex per pod.
type Coordinator struct {
	cfg      Config
	database db.Database
	ch       clickhouse.ClickHouse
	factory  *Factory
}

// New constructs a Coordinator. It uses the package-level db.Query singleton
// for queries and resolves read/write replicas through the injected
// db.Database, matching the pattern in svc/ctrl/worker.
func New(cfg Config, database db.Database, ch clickhouse.ClickHouse, factory *Factory) *Coordinator {
	return &Coordinator{
		cfg:      cfg,
		database: database,
		ch:       ch,
		factory:  factory,
	}
	//nolint:exhaustruct // groupOffset starts at zero by design.
	return &Coordinator{
		cfg:            cfg,
		database:       database,
		ch:             ch,
		factory:        factory,
		drainListCache: drainListCache,
	}, nil
}

// Run blocks until ctx is cancelled, ticking once every PollInterval.
// repeat.Every fires once immediately so metrics start populating without
// waiting a full interval (useful during a fresh deploy's readiness
// window) and then on the regular cadence. A missed tick — e.g. because
// the previous tick is still running on a large batch — is dropped, since
// each tick re-reads the cursor and queueing buys nothing.
func (c *Coordinator) Run(ctx context.Context) error {
	stop := repeat.Every(c.cfg.PollInterval, func() {
		if err := c.tick(ctx); err != nil {
			logger.Warn("logdrain tick failed", "error", err.Error())
		}
	})
	defer stop()
	<-ctx.Done()
	return nil
}

func (c *Coordinator) tick(ctx context.Context) error {
	start := time.Now()
	defer func() {
		metrics.TickDuration.Observe(time.Since(start).Seconds())
	}()

	rows, err := db.Query.ListEnabledLogDrains(ctx, c.database.RO())
	if err != nil {
		return err
	}
	rows = FilterByShard(rows, c.cfg.ShardCount, c.cfg.ShardIndex)

	groups, err := BuildGroups(rows)
	if err != nil {
		return err
	}

	// Enforce group limits to prevent ClickHouse query fan-out
	if len(groups) > c.cfg.MaxGroupsPerShard {
		logger.Warn("group limit exceeded, processing subset",
			"groups", len(groups),
			"limit", c.cfg.MaxGroupsPerShard,
			"shard", c.cfg.ShardIndex,
		)
		observability.GroupsSkippedLimit.WithLabelValues(strconv.Itoa(c.cfg.ShardIndex)).
			Add(float64(len(groups) - c.cfg.MaxGroupsPerShard))
		groups = groups[:c.cfg.MaxGroupsPerShard]
	}

	// Update active group metrics
	observability.ActiveGroups.WithLabelValues(strconv.Itoa(c.cfg.ShardIndex)).
		Set(float64(len(groups)))

	logger.Debug("logdrain tick", 
		"groups", len(groups), 
		"drains", len(rows),
		"shard", c.cfg.ShardIndex,
	)

	// Validate workspace drain limits for future drain creation enforcement
	workspaceDrainCounts := make(map[string]int)
	for _, row := range rows {
		workspaceDrainCounts[row.WorkspaceID]++
		if workspaceDrainCounts[row.WorkspaceID] > c.cfg.MaxDrainsPerWorkspace {
			logger.Warn("workspace exceeds drain limit",
				"workspace_id", row.WorkspaceID,
				"drain_count", workspaceDrainCounts[row.WorkspaceID],
				"limit", c.cfg.MaxDrainsPerWorkspace,
			)
		}
	}

	// Per-group processing is the next stack: it owns the CH query, the
	// per-drain transform/sink fan-out, and the cursor advance under
	// optimistic locking. Pulling that into its own commit keeps the
	// integration test (which needs dockertest CH + MySQL) reviewable
	// alongside the logic that drives it.
	for _, g := range groups {
		logger.Debug("would process group",
			"group_key", string(g.Key),
			"workspace", g.Workspace,
			"project", g.Project,
			"env", g.Env,
			"source", string(g.Source),
			"drains", len(g.Drains),
		)
	}

	return nil
}

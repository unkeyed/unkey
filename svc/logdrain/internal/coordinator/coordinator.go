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

	"golang.org/x/sync/errgroup"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
)

// tickGroupWorkers caps how many groups are processed in parallel per
// tick. Each worker holds one ClickHouse connection plus the in-flight
// HTTP fan-out for its drains, so the value trades CH/sink concurrency
// against connection-pool pressure. 16 keeps a single replica useful up
// to ~500 groups per tick at sub-poll-interval latency without blowing
// past Go's default http.Transport / CH driver pool sizes.
const tickGroupWorkers = 16

// drainListTTL is the freshness window for the cached
// ListEnabledLogDrains result. Drains rarely change relative to the poll
// interval, so a small TTL collapses N pollers' MySQL load to roughly
// 1/TTL queries per second per replica without making a newly-created
// drain wait noticeably longer than today's poll_interval to start
// draining.
const drainListTTL = 5 * time.Second

// Config carries the policy knobs the coordinator reads from svc/logdrain.
//
// Ordinal is this pod's StatefulSet index (parsed from $HOSTNAME) and
// is used only as a low-cardinality metric label so dashboards can
// split per-pod. ShardStart/ShardEnd is the half-open range of shard
// buckets this pod owns out of TotalShards; FilterByShard uses the
// range to decide which workspaces to process.
type Config struct {
	PollInterval          time.Duration
	BatchWindow           time.Duration
	MaxBatchRecords       int
	PauseAfterFailures    int
	MaxGroupsPerShard     int
	MaxDrainsPerWorkspace int
	Ordinal               int
	ShardStart            int
	ShardEnd              int
}

// drainListCacheKey is the only key used in drainListCache. The cache is
// effectively a single-slot SWR on the global ListEnabledLogDrains
// query; cache.Cache is keyed on a generic K, so we pass an empty string
// (every tick reads the same slot).
const drainListCacheKey = ""

// Coordinator wires the loaded drains, the ClickHouse client, the sink
// factory, and the MySQL queries together. v1 runs a single instance per
// region; sharding is wired via Config so a future multi-replica rollout
// only needs the helm chart to set Replicas; the per-pod ordinal and
// shard range are derived from $HOSTNAME at startup.
type Coordinator struct {
	cfg      Config
	database db.Database
	ch       clickhouse.ClickHouse
	factory  *Factory

	// drainListCache memoises ListEnabledLogDrains via stale-while-
	// revalidate so a thundering tick collapses to one MySQL read per
	// drainListTTL even across future concurrent callers (admin RPCs,
	// readiness probes). Single-key cache, since the query takes no
	// arguments.
	drainListCache cache.Cache[string, []db.ListEnabledLogDrainsRow]

	// groupOffset rotates the starting index of the per-tick group
	// window when len(groups) > MaxGroupsPerShard, so over-capacity
	// shards round-robin through every group instead of permanently
	// starving the lexicographic tail. Mutated only inside tick(),
	// which is single-goroutine, so no synchronization is needed.
	groupOffset int
}

// New constructs a Coordinator. It uses the package-level db.Query singleton
// for queries and resolves read/write replicas through the injected
// db.Database, matching the pattern in svc/ctrl/worker.
func New(cfg Config, database db.Database, ch clickhouse.ClickHouse, factory *Factory) (*Coordinator, error) {
	drainListCache, err := cache.New(cache.Config[string, []db.ListEnabledLogDrainsRow]{
		// Stale = 2 × Fresh gives one full TTL of grace under MySQL
		// hiccups: when the primary is briefly unreachable, the next
		// tick keeps draining against the last good drain list instead
		// of stalling the whole pod.
		Fresh:    drainListTTL,
		Stale:    2 * drainListTTL,
		MaxSize:  1,
		Resource: "logdrain_enabled_drains",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("create drain list cache: %w", err)
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

// listEnabledDrains returns the cached set of enabled drains, refreshing
// asynchronously when the entry is missing or stale. SWR keeps the read
// path lock-free in steady state — the typical tick hits a fresh cache
// and never touches MySQL.
func (c *Coordinator) listEnabledDrains(ctx context.Context) ([]db.ListEnabledLogDrainsRow, error) {
	rows, _, err := c.drainListCache.SWR(ctx, drainListCacheKey,
		func(ctx context.Context) ([]db.ListEnabledLogDrainsRow, error) {
			return db.Query.ListEnabledLogDrains(ctx, c.database.RO())
		},
		func(refreshErr error) cache.Op {
			if refreshErr != nil {
				return cache.Noop
			}
			return cache.WriteValue
		},
	)
	return rows, err
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

	rows, err := c.listEnabledDrains(ctx)
	if err != nil {
		return err
	}
	rows = FilterByShard(rows, c.cfg.ShardStart, c.cfg.ShardEnd)

	groups, err := BuildGroups(rows)
	if err != nil {
		return err
	}

	// Enforce group limits to prevent ClickHouse query fan-out. When the
	// shard is over capacity we process a rotating window of size
	// MaxGroupsPerShard, advancing groupOffset by the window size each
	// tick. Every group still gets serviced within
	// ceil(len(groups)/MaxGroupsPerShard) ticks instead of being
	// permanently starved by the deterministic sort order.
	if len(groups) > c.cfg.MaxGroupsPerShard {
		logger.Warn("group limit exceeded, rotating subset",
			"groups", len(groups),
			"limit", c.cfg.MaxGroupsPerShard,
			"offset", c.groupOffset,
			"shard", c.cfg.Ordinal,
		)
		metrics.GroupsSkippedLimit.WithLabelValues(strconv.Itoa(c.cfg.Ordinal)).
			Add(float64(len(groups) - c.cfg.MaxGroupsPerShard))

		offset := c.groupOffset % len(groups)
		end := offset + c.cfg.MaxGroupsPerShard
		if end <= len(groups) {
			groups = groups[offset:end]
		} else {
			// Window wraps around the end of the slice; build a new
			// contiguous slice rather than try to teach the worker pool
			// about a two-segment view.
			wrapped := make([]Group, 0, c.cfg.MaxGroupsPerShard)
			wrapped = append(wrapped, groups[offset:]...)
			wrapped = append(wrapped, groups[:end-len(groups)]...)
			groups = wrapped
		}
		c.groupOffset += c.cfg.MaxGroupsPerShard
	} else {
		// Reset the offset so it doesn't grow unbounded once the shard
		// drops back under capacity.
		c.groupOffset = 0
	}

	// Update active group metrics
	metrics.ActiveGroups.WithLabelValues(strconv.Itoa(c.cfg.Ordinal)).
		Set(float64(len(groups)))

	logger.Debug("logdrain tick",
		"groups", len(groups),
		"drains", len(rows),
		"shard", c.cfg.Ordinal,
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

	// Index the rows so the per-group fan-out can resolve a drain's full
	// log_drains row (credentials, provider, config) without re-querying
	// MySQL inside the inner loop.
	drainsByID := make(map[string]db.ListEnabledLogDrainsRow, len(rows))
	for _, row := range rows {
		drainsByID[row.ID] = row
	}

	// Bounded concurrent group processing. Serial fan-out makes each
	// tick proportional to len(groups) × per-group latency, which blows
	// past poll_interval well before MaxGroupsPerShard. With a worker
	// pool the tick latency is bounded by ceil(len(groups)/workers) ×
	// per-group latency, recovering the headroom the architecture
	// document promised. errgroup is used purely for the WaitGroup +
	// concurrency-limit semantics; per-group failures are already
	// logged inside the goroutine, so we never return an error.
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(tickGroupWorkers)
	for _, g := range groups {
		eg.Go(func() error {
			if err := c.processGroup(gctx, g, drainsByID); err != nil {
				logger.Warn("process group failed",
					"group_key", string(g.Key),
					"error", err.Error(),
				)
			}
			return nil
		})
	}
	_ = eg.Wait()

	return nil
}

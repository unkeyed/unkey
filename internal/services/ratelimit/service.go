package ratelimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/buffer"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"

	"github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// DB is the contract Config.DB must satisfy: a primary/replica pair backed
// by [*mysql.Replica]. The standard pkg/mysql.New() result satisfies this
// directly, so callers don't need to wrap or adapt anything; they pass the
// shared application database in and the ratelimit service constructs its
// own typed query layer internally.
type DB interface {
	RW() *mysql.Replica
	RO() *mysql.Replica
}

// ErrDBRequired is returned by [New] when Config.DB is nil. The service
// requires a database connection for cross-region propagation; a nil DB is
// always a configuration bug.
var ErrDBRequired = errors.New("ratelimit: Config.DB is required")

// maxCASRetries bounds every CAS loop in the package to prevent livelock
// under pathological contention. Under normal load, CAS retries are rare and
// bounded by GOMAXPROCS; 100 is astronomically generous.
const maxCASRetries = 100

// atomicMax raises target to val atomically via a bounded CAS loop.
// The value only ever moves forward — a smaller val is a no-op. If the retry
// bound is exhausted, an error is logged and the function returns without
// updating; callers that depend on monotonic progress must tolerate stale
// values in that case.
func atomicMax(target *atomic.Int64, val int64) {
	for range maxCASRetries {
		cur := target.Load()
		if val <= cur {
			return
		}
		if target.CompareAndSwap(cur, val) {
			return
		}
	}
	logger.Error("atomicMax retries exhausted, proceeding with stale counter")
}

// counterEntry pairs a sliding-window counter with a sync.Once that gates
// origin hydration. The fetch closure is bound at entry creation so the
// hot path doesn't need to allocate a per-call closure to pass into
// Hydrate. The hydrated flag is a fast-path optimization: once set,
// callers skip sync.Once.Do entirely. The sync.Once still enforces
// correctness on the first call — concurrent callers on a cold entry
// block inside Do until the first fetch returns, closing the race where
// LoadOrStore would hand out a zero-valued counter to late arrivals
// before the owner finished hydrating it.
//
// blocked gates the cross-region propagation event for this counter's
// (workspace, namespace, identifier, duration, sequence) tuple. Set true
// exactly once per entry: either by activateStrictMode CAS-ing it when the
// local denial fires the originating event, or by the blocklist sync
// goroutine when it inflates the counter from a row that some other region
// already wrote. Once true, subsequent denials on this entry are known to be
// either self-driven or sync-driven echoes, and the propagation path skips
// them.
type counterEntry struct {
	val      atomic.Int64
	once     sync.Once
	hydrated atomic.Bool
	blocked  atomic.Bool
	fetch    func(context.Context) int64
}

// Hydrate populates val from the bound fetcher exactly once. The fetched
// value is CAS-merged via atomicMax so the local counter only moves
// forward. Concurrent callers on a cold entry block inside Do until the
// first fetch returns; subsequent callers return after a single atomic
// load of hydrated. A fetch failure surfaces as a returned 0, which
// atomicMax treats as a no-op — the entry is left at whatever val held
// before (typically 0 on a failed first hydration) and marked hydrated
// so the hot path stays fast.
func (e *counterEntry) Hydrate(ctx context.Context) {
	if e.hydrated.Load() {
		return
	}
	e.once.Do(func() {
		atomicMax(&e.val, e.fetch(ctx))
		e.hydrated.Store(true)
	})
}

// service implements lockless distributed rate limiting using a sliding window
// algorithm with atomic counters.
//
// All rate limit state is stored in two flat sync.Maps with no mutexes in
// the hot path:
//
//   - counters: per-window counter entries keyed by (workspace, namespace,
//     identifier, duration, sequence). Each entry holds the window's request
//     count plus a sync.Once coordinating the first origin hydration.
//   - strictUntils: per-identifier deadlines keyed by (workspace, namespace,
//     identifier, duration). Set when a request is denied; subsequent
//     requests on the same identifier force an origin fetch until the
//     deadline passes.
//
// Keeping them in separate maps means strictUntil's lifecycle is decoupled
// from any single window's sequence, which matches the semantics: a denial
// in window N triggers strict enforcement that can extend into window N+1.
//
// Ratelimit uses a CAS loop: denials are wait-free (single atomic Load), allows
// retry only when another goroutine incremented the same counter between Load
// and CAS. RatelimitMany uses optimistic Add with rollback for atomic
// all-or-nothing batch semantics.
//
// Local counters are eventually consistent with Redis. Background replay workers
// push local increments to Redis and CAS-merge the global count back into the
// local atomic counter.
//
// Denials also propagate cross-region: a strict-mode transition writes one
// row to ratelimit_blocklist (batched, circuit-broken), and a periodic sync
// goroutine reads the active set and inflates local counters so other regions
// deny the same identifier without seeing its traffic firsthand. See
// blocklist.go.
type service struct {
	clock clock.Clock

	// counters maps counterKey -> *atomic.Int64.
	// Each entry is one sliding window's request count for a specific
	// (workspace, namespace, identifier, duration) combination.
	counters sync.Map

	// strictUntils maps strictKey -> *atomic.Int64 storing unix millis.
	// When non-zero and in the future, Ratelimit forces a synchronous origin
	// fetch for the keyed (workspace, namespace, identifier, duration) tuple
	// before deciding, trading latency for tighter convergence after a denial.
	strictUntils sync.Map

	// origin is the distributed source-of-truth counter (typically Redis).
	// Local atomics in `counters` converge toward origin via async replay.
	origin counter.Counter

	// replayBuffer holds rate limit events for async propagation to Redis.
	replayBuffer *buffer.Buffer[RatelimitRequest]

	// originCircuitBreaker wraps every call to the origin counter (both
	// replay INCR and cold/strict-mode GET). When tripped, requests use
	// whatever local state is available rather than blocking on Redis.
	originCircuitBreaker circuitbreaker.CircuitBreaker[int64]

	// db is the cross-region propagation backend for denials. Constructed
	// inside [New] from Config.DB. Always non-nil; Config.DB is required.
	db *db.Database

	// blocklistWriter batches propagation rows flushed on the cold→hot
	// transition in activateStrictMode. Drops on full buffer rather than
	// block the hot path; the same denial will retry on the next transition
	// anyway. Element type is the sqlc-generated insert params directly so
	// the flush has nothing to translate.
	blocklistWriter *batch.BatchProcessor[db.BlocklistInsertParams]

	// blocklistCircuitBreaker wraps the batched MySQL upsert. When tripped,
	// the flush logs and discards the batch so a sick database does not
	// stall the batch processor or back-pressure denials.
	blocklistCircuitBreaker circuitbreaker.CircuitBreaker[any]
}

// Config holds configuration for creating a new rate limiting service.
type Config struct {
	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	Counter counter.Counter

	// DB drives cross-region propagation of denials: the service writes to
	// ratelimit_blocklist on each strict-mode transition and periodically
	// reads the active set to inflate local counters for identifiers that
	// other regions have already denied. Required; [New] returns
	// [ErrDBRequired] if nil.
	//
	// The standard application database from pkg/db satisfies [DB] directly;
	// no adapter is needed.
	DB DB
}

// New creates a new rate limiting service.
//
// The service starts 8 background goroutines to process the replay buffer,
// synchronizing local rate limit state with Redis. It also starts a goroutine
// to periodically clean up expired counters, and a sync goroutine that pulls
// the cross-region blocklist from MySQL.
//
// Returns [ErrDBRequired] if config.DB is nil. Call Close when done to
// release resources.
func New(config Config) (*service, error) {
	if config.DB == nil {
		return nil, ErrDBRequired
	}
	if config.Clock == nil {
		config.Clock = clock.New()
	}

	s := &service{ //nolint:exhaustruct // blocklistWriter needs s.flushBlocklistBatch and is wired below
		clock:        config.Clock,
		counters:     sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		strictUntils: sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		origin:       config.Counter,
		replayBuffer: buffer.New[RatelimitRequest](buffer.Config{
			Name:     "ratelimit_replays",
			Capacity: 10_000,
			Drop:     true,
		}),
		originCircuitBreaker:    circuitbreaker.New[int64]("ratelimitOrigin"),
		blocklistCircuitBreaker: circuitbreaker.New[any]("ratelimit_blocklist_writes"),
		db:                      db.New(config.DB.RW(), config.DB.RO()),
	}
	s.blocklistWriter = batch.New[db.BlocklistInsertParams](batch.Config[db.BlocklistInsertParams]{
		Name:          "ratelimit_blocklist",
		Drop:          true,
		BatchSize:     100,
		BufferSize:    10_000,
		FlushInterval: time.Second,
		Consumers:     1,
		Flush:         s.flushBlocklistBatch,
	})
	s.startBlocklistSync()

	s.startJanitor()

	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Close stops the replay buffer and releases resources.
func (s *service) Close() error {
	s.replayBuffer.Close()
	s.blocklistWriter.Close()
	return nil
}

// loadCounter returns the counter entry for the given key, creating it if needed.
// Newly created entries carry a fetcher closure bound to the key, so callers
// only need to invoke entry.Hydrate(ctx) to ensure the counter has been
// populated from origin. Callers that skip Hydrate risk reading a zero-valued
// counter while another goroutine is mid-fetch.
//
// Counters created here are attributed to traffic via RatelimitWindowsCreated.
// The blocklist sync uses [findOrCreateCounter] directly so its insertions
// land on a separate metric and don't inflate the traffic-driven counter.
func (s *service) loadCounter(key counterKey) *counterEntry {
	entry, created := s.findOrCreateCounter(key)
	if created {
		metrics.RatelimitWindowsCreated.Inc()
	}
	return entry
}

// loadStrictUntil returns the unix-millis deadline for strict-enforcement
// mode on this (workspace, namespace, identifier, duration) tuple, or 0 if
// no deadline is set.
func (s *service) loadStrictUntil(key strictKey) int64 {
	val, ok := s.strictUntils.Load(key)
	if !ok {
		return 0
	}
	return val.(*atomic.Int64).Load()
}

// setStrictUntil raises the strict-enforcement deadline to untilMs, atomically.
// The deadline only ever moves forward — a concurrent caller setting an earlier
// deadline is a no-op. Every call counts as a strict-mode activation for
// observability, regardless of whether the deadline actually advanced.
func (s *service) setStrictUntil(key strictKey, untilMs int64) {
	val, _ := s.strictUntils.LoadOrStore(key, &atomic.Int64{})
	atomicMax(val.(*atomic.Int64), untilMs)
	metrics.RatelimitStrictModeActivations.Inc()
}

// minPropagationDuration is the shortest window length for which cross-region
// propagation is meaningful. The pipeline takes up to one batch flush (1s)
// plus one sync interval (10s) to reach receivers; below this floor the
// originating window has rotated by the time the row arrives, so propagation
// is pure write/read load with no enforcement value. Local strict mode still
// fires regardless of window length.
const minPropagationDuration = time.Minute

// activateStrictMode is the single denial-side path: raise the local
// strict-mode deadline and, if this counter entry has not yet been blocked,
// buffer a propagation event. The entry's blocked flag is the dedup
// primitive: it's CAS'd false→true here on a fresh denial, and the blocklist
// sync goroutine pre-sets it to true on rows it inflates from MySQL. So
// sync-inflated counters never re-emit their own state back to the
// propagation channel, while every (region, sequence) that originates a
// denial emits exactly once.
//
// expires_at on the propagation row is derived from sequence rather than
// from req.Time: the row only matters while the inflated counter still
// affects sliding-window math, which ends at the close of the window after
// the originating one (cur in S, prev in S+1, gone after S+1). Using
// (sequence+2)*duration makes expires_at deterministic; every emit for the
// same sequence computes the same value, so receiver-driven echoes never
// "extend" anything in MySQL.
//
// Two filters gate emission. Window durations below [minPropagationDuration]
// rotate before the row reaches receivers, so we skip them. And denials
// where the user has consumed less than half their limit before this
// request's cost are skipped too; those typically come from a single
// oversized request and broadcasting them would block other regions for a
// user who still has plenty of legitimate quota left. The threshold stops at
// half rather than at full so the (used=8, cost=3, limit=10) case still
// propagates: the user is heavily consuming and demanding more than they
// have; over-blocking the last 2 credits in other regions is a fair
// trade-off against letting them spread cost>1 traffic across regions
// undetected.
//
// effectiveCount is the post-cost sliding-window count the caller already
// computed for the deny decision; we pass it in to avoid re-running the
// math. preCost = effectiveCount - req.Cost since slidingWindowCount is
// linear in curCount.
//
// Both Ratelimit (single-key CAS denial) and RatelimitMany (per-entry rollback
// after a batch failure) call this; keeping the propagation shape in one place
// avoids the two paths drifting on what gets written to the blocklist.
func (s *service) activateStrictMode(req RatelimitRequest, cs *checkState, effectiveCount int64) {
	durationMs := req.Duration.Milliseconds()
	s.setStrictUntil(cs.strictKey, req.Time.UnixMilli()+durationMs)

	if req.Duration < minPropagationDuration {
		return
	}

	preCost := effectiveCount - req.Cost
	if 2*preCost < req.Limit {
		return
	}

	if !cs.cur.blocked.CompareAndSwap(false, true) {
		return
	}
	s.blocklistWriter.Buffer(db.BlocklistInsertParams{
		WorkspaceID: req.WorkspaceID,
		Namespace:   req.Namespace,
		Identifier:  req.Identifier,
		DurationMs:  uint64(durationMs),
		Sequence:    cs.curSequence,
		Limit:       uint64(req.Limit),
		ExpiresAt:   uint64((cs.curSequence + 2) * durationMs),
	})
}

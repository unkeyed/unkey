package ratelimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

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

// ErrRegionRequired is returned by [New] when Config.Region is empty. The
// region tag partitions every cross-region write so receivers can sum
// contributions across regions; an empty region would collapse all rows
// into a single ambiguous bucket.
var ErrRegionRequired = errors.New("ratelimit: Config.Region is required")

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

// counterEntry holds the in-memory state for one sliding-window cell:
// a (workspace, namespace, identifier, duration, sequence) tuple. One
// entry per active cell; entries are evicted by the janitor after the
// window has fully aged out.
type counterEntry struct {
	// val is this region's own observed count for the cell, populated by
	// passing requests in the CAS loop and CAS-merged from Redis via the
	// background replay path. The deny decision adds val to
	// globalCount to get the global count.
	val atomic.Int64

	// once gates the first call to fetch so concurrent callers on a cold
	// entry block inside Do until hydration returns, closing the race
	// where LoadOrStore would hand out a zero-valued counter to late
	// arrivals before the owner finished fetching from origin.
	once sync.Once

	// hydrated is a fast-path optimization: once set, callers skip
	// sync.Once.Do entirely. The sync.Once still enforces correctness on
	// the first call.
	hydrated atomic.Bool

	// globalCount is the sum of other regions' counts for this cell
	// as of the most recent cross-region sync. The deny decision uses
	// cur.val + cur.globalCount plus the prev-window analogue, so
	// this region's local observation is augmented with the cross-region
	// picture without inflating val itself — inflating val would feed
	// back into our own MySQL row on the next flush and double-count
	// other regions' contributions. Updated by the cross-region sync
	// goroutine via atomicMax; the request path only reads it.
	globalCount atomic.Int64

	// limit is the most recently observed per-request limit on this
	// entry, written by prepareCheck on every request that touches the
	// entry. The cross-region flush goroutine compares val/limit against
	// the utilization floor to skip writes for entries that haven't
	// crossed the propagation threshold. Different requests on the same
	// identifier may carry different limits in principle; last writer
	// wins, since the most recent request's view of the limit is the
	// closest-to-current value the flush goroutine can act on.
	limit atomic.Int64

	// lastFlushed is the val written by the previous successful
	// cross-region flush. The flush goroutine skips entries whose val
	// hasn't moved since the last flush, so quiet entries don't generate
	// redundant MySQL writes. Updated only by the flush goroutine, only
	// after the bulk upsert succeeds; a transient MySQL failure leaves
	// entries eligible for retry on the next tick.
	lastFlushed atomic.Int64

	// fetch is bound to the entry's key at creation so the hot path
	// doesn't allocate a per-call closure to pass into Hydrate.
	fetch func(context.Context) int64
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

// service implements lockless distributed rate limiting using a sliding
// window algorithm with atomic counters. All rate limit state is held in a
// single flat sync.Map with no mutexes in the hot path: per-window counter
// entries keyed by (workspace, namespace, identifier, duration, sequence),
// each holding the window's request count plus a sync.Once coordinating the
// first origin hydration.
//
// Ratelimit uses a CAS loop: denials are wait-free (single atomic Load),
// allows retry only when another goroutine incremented the same counter
// between Load and CAS. RatelimitMany uses optimistic Add with rollback for
// atomic all-or-nothing batch semantics.
//
// Local counters are eventually consistent with Redis. Background replay
// workers push local increments to Redis and CAS-merge the global count
// back into the local atomic counter.
//
// Cross-region awareness comes from ratelimit_window_counts: each region
// periodically flushes its own count for each active window into a row
// keyed by (workspace, namespace, identifier, duration, sequence, region),
// and a periodic sync sums other regions' rows into counterEntry.globalCount.
// "Global" throughout this package means "across all other regions" —
// not "across all nodes". Nodes within a region already converge through
// Redis replay; global state excludes own-region rows on read. The
// sliding-window math includes both the local val and the global sum, so
// denials in region B reflect what region A has already seen without B
// having processed that traffic. See global.go.
type service struct {
	clock clock.Clock

	// counters maps counterKey -> *atomic.Int64.
	// Each entry is one sliding window's request count for a specific
	// (workspace, namespace, identifier, duration) combination.
	counters sync.Map

	// strictUntils maps strictKey -> *atomic.Int64 storing unix millis.
	// When non-zero and in the future, Ratelimit forces a synchronous
	// origin fetch for the keyed (workspace, namespace, identifier,
	// duration) tuple before deciding, trading latency for tighter
	// convergence after a denial. The deadline is keyed without sequence
	// because a denial in window N triggers strict enforcement that may
	// extend into window N+1.
	//
	// This is the in-region convergence mechanism: instances within a
	// region share state through Redis, and a forced fetch on the request
	// path drains any lag between an instance's local view and the
	// region's Redis-backed truth. Cross-region convergence is a separate
	// concern handled by ratelimit_window_counts.
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

	// db is the cross-region propagation backend. Constructed inside [New]
	// from Config.DB. Always non-nil; Config.DB is required.
	db *db.Database

	// region tags every row this service writes to ratelimit_window_counts
	// and is the predicate the sync loop uses to filter out own-region rows
	// when reading. Constant for the lifetime of the service.
	region string

	// globalCircuitBreaker wraps the periodic bulk upsert for the
	// cross-region ratelimit_window_counts table. When tripped, the flush
	// logs and discards the batch so a sick database does not stall
	// subsequent ticks or back-pressure denials on the request path.
	globalCircuitBreaker circuitbreaker.CircuitBreaker[any]
}

// Config holds configuration for creating a new rate limiting service.
type Config struct {
	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	Counter counter.Counter

	// DB drives cross-region count sharing via ratelimit_window_counts:
	// each region periodically flushes its own observed count for active
	// windows, and a sync goroutine reads the rows from other regions and
	// folds the sum into local counterEntry.globalCount. Required;
	// [New] returns [ErrDBRequired] if nil.
	//
	// The standard application database from pkg/db satisfies [DB]
	// directly; no adapter is needed.
	DB DB

	// Region identifies the running fleet for cross-region count sharing.
	// Sourced from UNKEY_REGION at process start. Used as the row-key
	// partition for own writes to ratelimit_window_counts and the filter
	// for skipping own-region rows on read. Required; [New] returns
	// [ErrRegionRequired] if empty.
	Region string
}

// New creates a new rate limiting service.
//
// The service starts 8 background goroutines to process the replay buffer,
// synchronizing local rate limit state with Redis. It also starts a janitor
// to clean up expired counters, plus a flush goroutine that emits per-region
// counts to ratelimit_window_counts and a sync goroutine that pulls and sums
// other regions' contributions.
//
// Returns [ErrDBRequired] if config.DB is nil or [ErrRegionRequired] if
// config.Region is empty. Call Close when done to release resources.
func New(config Config) (*service, error) {
	if config.DB == nil {
		return nil, ErrDBRequired
	}
	if config.Region == "" {
		return nil, ErrRegionRequired
	}
	if config.Clock == nil {
		config.Clock = clock.New()
	}

	s := &service{
		clock:        config.Clock,
		counters:     sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		strictUntils: sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		origin:       config.Counter,
		region:       config.Region,
		replayBuffer: buffer.New[RatelimitRequest](buffer.Config{
			Name:     "ratelimit_replays",
			Capacity: 10_000,
			Drop:     true,
		}),
		originCircuitBreaker: circuitbreaker.New[int64]("ratelimitOrigin"),
		globalCircuitBreaker: circuitbreaker.New[any]("ratelimit_global_writes"),
		db:                   db.New(config.DB.RW(), config.DB.RO()),
	}
	s.startGlobalFlush()
	s.startGlobalSync()

	s.startJanitor()

	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Close stops the replay buffer and releases resources.
func (s *service) Close() error {
	s.replayBuffer.Close()
	return nil
}

// loadCounter returns the counter entry for the given key, creating it if
// needed. Newly created entries carry a fetcher closure bound to the key,
// so callers only need to invoke entry.Hydrate(ctx) to ensure the counter
// has been populated from origin. Callers that skip Hydrate risk reading a
// zero-valued counter while another goroutine is mid-fetch.
//
// Counters created here are attributed to traffic via RatelimitWindowsCreated.
// The window-counts sync uses [findOrCreateCounter] directly so its
// insertions land on a separate metric and don't inflate the traffic-driven
// counter.
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

// setStrictUntil raises the strict-enforcement deadline to untilMs,
// atomically. The deadline only ever moves forward; a concurrent caller
// setting an earlier deadline is a no-op. Every call counts as a
// strict-mode activation for observability, regardless of whether the
// deadline actually advanced.
func (s *service) setStrictUntil(key strictKey, untilMs int64) {
	val, _ := s.strictUntils.LoadOrStore(key, &atomic.Int64{})
	atomicMax(val.(*atomic.Int64), untilMs)
	metrics.RatelimitStrictModeActivations.Inc()
}

// findOrCreateCounter returns the entry for the given key, creating one if
// missing. Reports whether the entry was newly inserted so callers can
// attribute the creation to traffic vs. cross-region sync without polluting
// each other's metrics. Does not touch any metric itself.
func (s *service) findOrCreateCounter(key counterKey) (*counterEntry, bool) {
	if v, ok := s.counters.Load(key); ok {
		return v.(*counterEntry), false
	}
	fresh := &counterEntry{ //nolint:exhaustruct // other fields zero-initialize correctly
		fetch: func(ctx context.Context) int64 { return s.fetchFromOrigin(ctx, key) },
	}
	actual, loaded := s.counters.LoadOrStore(key, fresh)
	return actual.(*counterEntry), !loaded
}

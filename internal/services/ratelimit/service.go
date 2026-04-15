package ratelimit

import (
	"sync"
	"sync/atomic"

	"github.com/unkeyed/unkey/pkg/buffer"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/logger"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

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
// origin hydration. Only the goroutine that inserted the entry runs the
// first fetchFromOrigin; concurrent callers block inside Do until it
// returns, then take the fast path (atomic load on Once.done) forever.
// This closes the race where LoadOrStore would hand out a zero-valued
// counter to late arrivals before the owner finished hydrating it.
type counterEntry struct {
	val  atomic.Int64
	once sync.Once
}

// service implements lockless distributed rate limiting using a sliding window
// algorithm with atomic counters.
//
// All rate limit state is stored in two flat sync.Maps with no mutexes in
// the hot path:
//
//   - counters: per-window counter entries keyed by (name, identifier,
//     duration, sequence). Each entry holds the window's request count plus
//     a sync.Once coordinating the first origin hydration.
//   - strictUntils: per-identifier deadlines keyed by (name, identifier,
//     duration). Set when a request is denied; subsequent requests on the
//     same identifier force an origin fetch until the deadline passes.
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
type service struct {
	clock clock.Clock

	// counters maps counterKey -> *atomic.Int64.
	// Each entry is one sliding window's request count for a specific
	// (name, identifier, duration) combination.
	counters sync.Map

	// strictUntils maps strictKey -> *atomic.Int64 storing unix millis.
	// When non-zero and in the future, Ratelimit forces a synchronous origin
	// fetch for the keyed (name, identifier, duration) tuple before deciding,
	// trading latency for tighter convergence after a denial.
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
}

// Config holds configuration for creating a new rate limiting service.
type Config struct {
	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	Counter counter.Counter
}

// New creates a new rate limiting service.
//
// The service starts 8 background goroutines to process the replay buffer,
// synchronizing local rate limit state with Redis. It also starts a goroutine
// to periodically clean up expired counters.
//
// Call Close when done to release resources.
func New(config Config) (*service, error) {
	if config.Clock == nil {
		config.Clock = clock.New()
	}

	s := &service{
		clock:        config.Clock,
		counters:     sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		strictUntils: sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		origin:       config.Counter,
		replayBuffer: buffer.New[RatelimitRequest](buffer.Config{
			Name:     "ratelimit_replays",
			Capacity: 10_000,
			Drop:     true,
		}),
		originCircuitBreaker: circuitbreaker.New[int64]("ratelimitOrigin"),
	}

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

// loadCounter returns the counter entry for the given key, creating it if needed.
// Callers must invoke entry.once.Do(...) before reading entry.val to ensure the
// counter has been hydrated from origin — otherwise late arrivals on a cold key
// would observe a zero-valued counter before the first caller finishes its
// Redis fetch.
func (s *service) loadCounter(key counterKey) *counterEntry {
	if v, ok := s.counters.Load(key); ok {
		return v.(*counterEntry)
	}
	fresh := &counterEntry{} //nolint:exhaustruct // zero values are correct
	actual, loaded := s.counters.LoadOrStore(key, fresh)
	if !loaded {
		metrics.RatelimitWindowsCreated.Inc()
	}
	return actual.(*counterEntry)
}

// loadStrictUntil returns the unix-millis deadline for strict-enforcement
// mode on this (name, identifier, duration) tuple, or 0 if no deadline is set.
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

package gcra

import (
	"context"
	"hash/maphash"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/gcra/metrics"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// NameMaxLength matches the ratelimit_gcra.name column. A longer name would
	// fail its whole persist batch.
	NameMaxLength = 512

	// peerCacheFresh trades visibility for load: other writers' changes show up
	// about a second late, and a busy name costs one replica query per second per
	// pod instead of one per request.
	peerCacheFresh = time.Second
	// peerCacheStale serves a stale peer state while it refreshes, so a slow
	// replica delays visibility instead of blocking requests.
	peerCacheStale = 5 * time.Second
	// peerCacheEntriesMax bounds the cache's memory. A name that falls out costs
	// one blocking read the next time it is used.
	peerCacheEntriesMax = 100_000
	// peerReadTimeout bounds a cold read, which blocks its caller.
	peerReadTimeout = 5 * time.Second

	// persistBatchRowsMax bounds one INSERT statement.
	persistBatchRowsMax = 10_000
	// persistTickRowsMax bounds the rows one flush writes. Entries left over
	// stay dirty and are written by the next flush.
	persistTickRowsMax = 100_000
	// persistFlushInterval is the longest a change waits before it is written.
	// It runs on the Limiter's clock, so tests that drive the clock see the same
	// visibility delay as production.
	persistFlushInterval = time.Second
	// persistTimeout bounds one persist with its retries.
	persistTimeout = 5 * time.Second
	// closeTimeout keeps a shutdown from hanging on a sick database.
	closeTimeout = 15 * time.Second
	// closeFlushesMax bounds the flushes Close runs to write every dirty entry.
	closeFlushesMax = 10

	// decideAttemptsMax bounds how often a call starts over because maintenance
	// forgot one of its entries between the peer read and the decision.
	decideAttemptsMax = 3

	shardCount = 64
)

// Database provides the connections the Limiter uses. [mysql.MySQL] satisfies it.
type Database interface {
	// RW returns the primary, used for persisting and cleanup.
	RW() *mysql.Replica

	// RO returns the replica, used to read the other writers' rows.
	RO() *mysql.Replica
}

// Config borrows the database and clock.
type Config struct {
	// DB provides the primary and replica connections. Required.
	DB Database

	// Clock drives decisions, caching, circuit breakers and maintenance. Required.
	Clock clock.Clock
}

// Request is one rate limit check.
type Request struct {
	// Name identifies the rate limit globally. It must include every scope of
	// the limit, such as tenant and duration, but not the limit itself, so a
	// limit change keeps what was spent. At most [NameMaxLength] bytes.
	Name string

	Rate Rate

	// Cost is the number of units to spend. Zero checks without spending.
	Cost int64

	// Time is when the request happened. If zero, the Limiter uses its clock.
	Time time.Time
}

// Limiter is a distributed GCRA rate limiter. Each Limiter is a set of writers,
// one per name it holds in memory, and shares state with other Limiters through
// MySQL. It is safe for concurrent use.
type Limiter struct {
	database Database
	clock    clock.Clock

	// processID is the random part of every writer id of this Limiter.
	processID string

	// sequence numbers the entries this Limiter creates. Every entry gets a new
	// writer id, because a rebuilt entry starting at zero under an old id would
	// be hidden by GREATEST on upsert.
	sequence atomic.Int64

	shardSeed maphash.Seed
	shards    [shardCount]*shard

	// peerCache holds the combined state of every row of a name except the
	// row of the entry's writer id. Keys include the writer id.
	peerCache cache.Cache[string, State]

	circuitBreakerRead  circuitbreaker.CircuitBreaker[State]
	circuitBreakerWrite circuitbreaker.CircuitBreaker[int64]

	done      chan struct{}
	closeOnce sync.Once
}

// shard holds the entries of a slice of the names. Its lock serializes every
// decision on those names.
type shard struct {
	mu      sync.Mutex
	entries map[string]*entry
}

// entry is this process's writer for one name.
type entry struct {
	writerID string

	// state is what this writer admitted, ahead of MySQL.
	state State

	// persisted is the newest state MySQL acknowledged. The entry is dirty while
	// it differs from state.
	persisted State

	// expiresAtNs is sent with every row, the writer's latest reset_at plus the
	// limit's duration.
	expiresAtNs int64

	// resetAtNs is the writer's latest view of reset_at, used for eviction.
	resetAtNs int64

	// changedAt is the last time this process created or changed the entry.
	changedAt time.Time
}

func (e *entry) dirty() bool {
	return e.state != e.persisted
}

// New starts a Limiter with a new process id and its background work. The
// database stays owned by the caller.
func New(config Config) (*Limiter, error) {
	if err := assert.All(
		assert.NotNil(config.DB, "gcra database is required"),
		assert.NotNil(config.Clock, "gcra clock is required"),
	); err != nil {
		return nil, fault.Wrap(err, fault.Internal("invalid gcra configuration"))
	}
	peerCache, err := cache.New(cache.Config[string, State]{
		Fresh:    peerCacheFresh,
		Stale:    peerCacheStale,
		MaxSize:  peerCacheEntriesMax,
		Resource: "gcra",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("create gcra peer cache"))
	}
	var shards [shardCount]*shard
	for i := range shards {
		shards[i] = &shard{mu: sync.Mutex{}, entries: make(map[string]*entry)}
	}
	l := &Limiter{
		database:            config.DB,
		clock:               config.Clock,
		processID:           uid.New("", 16),
		sequence:            atomic.Int64{},
		shardSeed:           maphash.MakeSeed(),
		shards:              shards,
		peerCache:           peerCache,
		circuitBreakerRead:  circuitbreaker.New[State]("gcra_read", circuitbreaker.WithClock(config.Clock)),
		circuitBreakerWrite: circuitbreaker.New[int64]("gcra_write", circuitbreaker.WithClock(config.Clock)),
		done:                make(chan struct{}),
		closeOnce:           sync.Once{},
	}
	l.runPersist()
	l.runMaintenance()
	return l, nil
}

// Allow decides one request. See [Limiter.AllowMany].
func (l *Limiter) Allow(ctx context.Context, req Request) (Result, error) {
	results, err := l.AllowMany(ctx, []Request{req})
	if err != nil {
		return Result{}, err
	}
	return results[0], nil
}

// AllowMany decides several requests all or nothing: the costs are spent only
// if every request fits. Each result reports whether its own request fits; a
// request that fits but was not spent because another did not reports its
// own cost as remaining. Names must be distinct.
//
// The decision is atomic on this Limiter. Other Limiters' spending reaches it
// about a second late, so Limiters together can over-admit by what has not
// reached each other yet; the over-admission is paid back afterwards.
func (l *Limiter) AllowMany(ctx context.Context, reqs []Request) ([]Result, error) {
	seen := make(map[string]struct{}, len(reqs))
	for _, req := range reqs {
		if err := assert.All(
			assert.NotEmpty(req.Name, "gcra name must not be empty"),
			assert.LessOrEqual(len(req.Name), NameMaxLength, "gcra name is too long"),
			assert.Greater(req.Rate.Limit, 0, "gcra limit must be greater than zero"),
			assert.Greater(req.Rate.Duration, 0, "gcra duration must be greater than zero"),
			assert.GreaterOrEqual(req.Cost, 0, "gcra cost must not be negative"),
		); err != nil {
			return nil, err
		}
		if _, duplicate := seen[req.Name]; duplicate {
			return nil, fault.New("gcra names must be distinct", fault.Internal("duplicate name "+req.Name))
		}
		seen[req.Name] = struct{}{}
	}

	for range decideAttemptsMax {
		writerIDs := l.ensureEntries(reqs)
		peers := make([]State, len(reqs))
		for i, req := range reqs {
			peer, err := l.readPeers(ctx, req.Name, writerIDs[i])
			if err != nil {
				return nil, err
			}
			peers[i] = peer
		}
		results, ok, err := l.decide(reqs, writerIDs, peers)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		return results, nil
	}
	return nil, fault.New("gcra entries evicted repeatedly", fault.Internal("an entry was forgotten during every attempt"))
}

// ensureEntries returns the writer id of each request's entry, creating
// entries that do not exist.
func (l *Limiter) ensureEntries(reqs []Request) []string {
	writerIDs := make([]string, len(reqs))
	now := l.clock.Now()
	for i, req := range reqs {
		s := l.shard(req.Name)
		s.mu.Lock()
		e, ok := s.entries[req.Name]
		if !ok {
			e = &entry{
				writerID:    l.processID + "_" + strconv.FormatInt(l.sequence.Add(1), 10),
				state:       State{SpentNs: 0, FreshStartNs: 0},
				persisted:   State{SpentNs: 0, FreshStartNs: 0},
				expiresAtNs: 0,
				resetAtNs:   now.UnixNano(),
				changedAt:   now,
			}
			s.entries[req.Name] = e
			metrics.MemoryEntries.Inc()
		}
		writerIDs[i] = e.writerID
		s.mu.Unlock()
	}
	return writerIDs
}

// decide holds the locks of every shard that owns a request, in ascending
// order so concurrent calls cannot deadlock, decides every request and commits
// all of them if all fit. ok is false if an entry was forgotten or replaced
// since its writer id was read, because the peer state was read for that id.
func (l *Limiter) decide(reqs []Request, writerIDs []string, peers []State) (results []Result, ok bool, err error) {
	owners := make([]int, len(reqs))
	locked := make([]int, 0, len(reqs))
	for i, req := range reqs {
		owners[i] = l.shardIndex(req.Name)
		if !slices.Contains(locked, owners[i]) {
			locked = append(locked, owners[i])
		}
	}
	slices.Sort(locked)
	for _, index := range locked {
		l.shards[index].mu.Lock()
	}
	defer func() {
		for _, index := range slices.Backward(locked) {
			l.shards[index].mu.Unlock()
		}
	}()

	entries := make([]*entry, len(reqs))
	for i, req := range reqs {
		e, exists := l.shards[owners[i]].entries[req.Name]
		if !exists || e.writerID != writerIDs[i] {
			return nil, false, nil
		}
		entries[i] = e
	}

	now := l.clock.Now()
	ats := make([]time.Time, len(reqs))
	decisions := make([]Decision, len(reqs))
	allAllowed := true
	for i, req := range reqs {
		ats[i] = req.Time
		if ats[i].IsZero() {
			ats[i] = now
		}
		if decisions[i], err = Decide(ats[i], req.Rate, req.Cost, entries[i].state, peers[i]); err != nil {
			return nil, false, err
		}
		allAllowed = allAllowed && decisions[i].Allowed
	}

	results = make([]Result, len(reqs))
	for i, req := range reqs {
		d := decisions[i]
		e := entries[i]
		results[i] = d.Result
		outcome := "denied"
		if d.Allowed {
			outcome = "passed"
		}
		metrics.DecisionsTotal.WithLabelValues(outcome).Inc()
		e.resetAtNs = max(e.resetAtNs, d.ResetAt.UnixNano())

		if !allAllowed {
			if d.Allowed {
				unspent, err := Decide(ats[i], req.Rate, 0, e.state, peers[i])
				if err != nil {
					return nil, false, err
				}
				results[i].Remaining = unspent.Remaining
			}
			continue
		}
		if d.Writer == e.state {
			continue
		}
		e.state = d.Writer
		e.expiresAtNs = max(e.expiresAtNs, saturatingAdd(d.ResetAt.UnixNano(), req.Rate.Duration.Nanoseconds()))
		e.changedAt = now
		if d.Clamped {
			metrics.ClampsTotal.Inc()
		}
	}
	return results, true, nil
}

func (l *Limiter) shardIndex(name string) int {
	return int(maphash.String(l.shardSeed, name) % uint64(len(l.shards)))
}

func (l *Limiter) shard(name string) *shard {
	return l.shards[l.shardIndex(name)]
}

// Close stops the background work and persists every dirty entry,
// best effort within one deadline. Calls made during or after Close change
// memory but may not be persisted. Close is safe to call more than once.
func (l *Limiter) Close() {
	l.closeOnce.Do(func() {
		close(l.done)
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		for range closeFlushesMax {
			if l.flushDirty(ctx) == 0 || ctx.Err() != nil {
				break
			}
		}
		l.peerCache.Close()
	})
}

package ratelimit

import (
	"bytes"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	mathrand "math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Limiter is a global rate limiter. Each pod runs one Limiter, and the pods
// share their state through MySQL.
//
// Implementations are safe for concurrent use.
type Limiter interface {
	// Allow decides one request. See AllowMany.
	Allow(context.Context, Request) (Result, error)

	// AllowMany decides several requests all or nothing: the units are spent
	// only if every request fits. Each result reports whether its own request
	// fits. Fingerprints must be distinct.
	//
	// The decision is atomic on this pod. Other pods' spending reaches it a
	// few seconds late, so pods together can over-admit by what has not
	// reached each other yet. The over-admission is paid back afterwards,
	// because the level decays from the true sum.
	AllowMany(context.Context, []Request) ([]Result, error)

	// Close stops the background work and writes every row that MySQL has
	// not caught up with. It returns an error if that write fails. Allow and
	// AllowMany return an error after Close.
	Close() error
}

// Config holds the dependencies of a [Limiter].
type Config struct {
	// Clock gives the time of every decision. It should stay within
	// `maxClockSkew` of the MySQL clock; cleanup records a metric when it
	// does not. Tests and simulations pass a `clock.TestClock`. Required.
	Clock clock.Clock

	// Store holds the rows that all pods share. Required.
	Store Store

	// Random generates the pod ID and cleanup jitter. Tests and simulations
	// pass a seeded one to make runs repeatable. Optional; the default is
	// seeded from crypto/rand. Two pods must never share a seed: equal pod
	// IDs would hide each other's units.
	Random *mathrand.Rand
}

// Request is one rate limit check.
type Request struct {
	// Fingerprint identifies the limit. All pods that decide on the same
	// limit must use the same fingerprint, and rows in MySQL are stored under
	// it. The caller builds it, usually as a SHA-256 hash, and is responsible
	// for three rules:
	//
	// Include every scope that separates two limits, for example the
	// workspace, the namespace or key, and the identifier. A missing scope
	// makes two limits share one level.
	//
	// Include `Rate.Duration` but not `Rate.Limit`. Two durations on the same
	// identifier are two limits, and the duration sets the decay and the
	// periods. A changed limit must keep what was already consumed.
	//
	// Use an unambiguous encoding, for example a length prefix on each field,
	// so that ("ab", "c") and ("a", "bc") give different fingerprints.
	// Fingerprints are stored, so a change to the encoding resets the state
	// of every limit once.
	Fingerprint [32]byte

	Rate Rate

	// Units is the number of units to spend. Zero checks without spending.
	Units int64
}

// Result is the decision on one [Request].
type Result struct {
	Allowed bool

	// Remaining is how many more units fit right now.
	Remaining int64

	// ResetAt is when the next unit fits. It is never before the time of the
	// request.
	ResetAt time.Time
}

const (
	shardCount = 64

	podIDLength = 16

	// maxClockSkew is the largest difference between a pod clock and the
	// MySQL clock that the limiter allows for. Cleanup deletes a row this
	// much after it faded, see `deleteAfterNs`, and counts passes beyond it
	// in `clockSkewExceededTotal`.
	maxClockSkew = 5 * time.Second

	flushInterval = time.Second
	flushTimeout  = 5 * time.Second

	// MySQL allows at most 65,535 placeholders in a prepared statement, and
	// `UpsertRow` has 5 per row.
	maxRowsPerWrite = 5_000

	// Peer rows younger than rowsFreshFor are used as they are. Older rows
	// are used while a background read refreshes them. A request waits for
	// a read only when the rows are rowsStaleFor or older, and then for at
	// most readWaitTimeout. At most readsInFlightMax reads run at once, so
	// many new identifiers cannot take every connection of the shared
	// replica pool; a read beyond that counts as failed.
	rowsFreshFor     = time.Second
	rowsStaleFor     = 5 * time.Second
	readTimeout      = time.Second
	readWaitTimeout  = 100 * time.Millisecond
	readsInFlightMax = 16

	// maxPeriodsPerRead bounds a read, which returns at most two rows per
	// period, see `Store.ReadRows`. A row lives for its period and then
	// until it fades, about `Duration × (14 + ln Limit)` after the period
	// starts, so a limit has 2 to 5 periods with rows. A read that reaches
	// the bound misses only the oldest periods, which weigh less than
	// e^-1000 of a unit.
	maxPeriodsPerRead = 100

	// Each pod cleans up every `cleanupInterval`, give or take 20%. A pass
	// deletes faded rows in batches of `rowsPerDelete`, at most
	// `maxDeletesPerCleanupPass` batches. While passes stop at that bound,
	// the next one starts after `cleanupBacklogInterval`.
	cleanupInterval          = time.Minute
	cleanupBacklogInterval   = time.Second
	cleanupTimeout           = 30 * time.Second
	rowsPerDelete            = 10_000
	maxDeletesPerCleanupPass = 10
)

const podIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type limiter struct {
	clock clock.Clock
	store Store

	// randomMu guards random, which is not safe for concurrent use.
	randomMu sync.Mutex
	random   *mathrand.Rand

	// podID names this pod's rows. It is random and never reused.
	podID string

	shards [shardCount]shard
	closed atomic.Bool

	// flushMu lets one flush run at a time, so only the flush that collected
	// a row marks it as committed.
	flushMu sync.Mutex

	// runInBackground runs background reads. It is `go f()`; the simulation
	// replaces it to choose when they run.
	runInBackground func(f func())

	// readSlots holds one value per running read, see `readsInFlightMax`.
	readSlots chan struct{}

	// stopLoops is nil if the flush and cleanup loops do not run. Close
	// calls it and waits for `loopsDone`.
	stopLoops context.CancelFunc
	loopsDone sync.WaitGroup
}

// shard holds the entries of the fingerprints whose first byte maps to it.
// mu guards entries and every entry in it.
type shard struct {
	mu      sync.Mutex
	entries map[[32]byte]*entry
}

// New returns a [Limiter] that flushes every second and cleans up about
// every minute. It is safe to call concurrently.
func New(config Config) (Limiter, error) {
	l, err := newLimiter(config)
	if err != nil {
		return nil, err
	}
	ctx, stopLoops := context.WithCancel(context.Background())
	l.stopLoops = stopLoops
	ticker := l.clock.NewTicker(flushInterval)
	l.loopsDone.Go(func() { l.flushUntilStopped(ctx, ticker) })
	l.loopsDone.Go(func() { l.cleanUpUntilStopped(ctx) })
	return l, nil
}

// newLimiter returns a limiter without the flush and cleanup loops. Tests
// and the simulation use it and call flush and cleanUp themselves, so the
// order of events follows their seed and not the Go scheduler. It is safe to
// call concurrently.
func newLimiter(config Config) (*limiter, error) {
	err := assert.All(
		assert.NotNil(config.Clock, "clock is required"),
		assert.NotNil(config.Store, "store is required"),
	)
	if err != nil {
		return nil, err
	}
	random := config.Random
	if random == nil {
		var seed [16]byte
		_, err = rand.Read(seed[:])
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("unable to seed the pod ID generator"))
		}
		random = mathrand.New(mathrand.NewPCG(binary.LittleEndian.Uint64(seed[:8]), binary.LittleEndian.Uint64(seed[8:])))
	}
	l := &limiter{
		clock:    config.Clock,
		store:    config.Store,
		randomMu: sync.Mutex{},
		random:   random,
		podID:    "",
		shards:   [shardCount]shard{},
		closed:   atomic.Bool{},
		flushMu:  sync.Mutex{},
		runInBackground: func(f func()) {
			go f()
		},
		readSlots: make(chan struct{}, readsInFlightMax),
		stopLoops: nil,
		loopsDone: sync.WaitGroup{},
	}
	for i := range l.shards {
		l.shards[i].entries = make(map[[32]byte]*entry)
	}
	id := make([]byte, podIDLength)
	for i := range id {
		id[i] = podIDAlphabet[random.IntN(len(podIDAlphabet))]
	}
	l.podID = string(id)
	return l, nil
}

// Allow is safe to call concurrently. The caller must not hold a shard lock.
func (l *limiter) Allow(ctx context.Context, request Request) (Result, error) {
	results, err := l.AllowMany(ctx, []Request{request})
	if err != nil {
		return Result{}, err
	}
	return results[0], nil
}

// AllowMany is safe to call concurrently. It takes the shard locks of its
// requests itself, so the caller must not hold a shard lock.
func (l *limiter) AllowMany(ctx context.Context, requests []Request) ([]Result, error) {
	err := l.validate(requests)
	if err != nil {
		return nil, err
	}
	l.refreshRows(ctx, requests)
	nowNs := l.clock.Now().UnixNano()

	// Locking in ascending shard order keeps two calls from deadlocking.
	shardIndexes := make([]int, 0, len(requests))
	for _, request := range requests {
		shardIndexes = append(shardIndexes, shardIndex(request.Fingerprint))
	}
	slices.Sort(shardIndexes)
	shardIndexes = slices.Compact(shardIndexes)
	for _, index := range shardIndexes {
		l.shards[index].mu.Lock()
	}
	defer func() {
		for _, index := range shardIndexes {
			l.shards[index].mu.Unlock()
		}
	}()
	// Checked again under the locks: Close sets `closed` before its final
	// flush takes these locks, so that flush sees every unit spent here.
	if l.closed.Load() {
		return nil, fault.New("the rate limiter is closed")
	}

	// First decide every request without spending, so a denied request means
	// that nothing is spent.
	entries := make([]*entry, len(requests))
	decisions := make([]decision, len(requests))
	allAllowed := true
	for i, request := range requests {
		e := l.entryOf(request.Fingerprint, request.Rate)
		decisions[i], err = e.check(nowNs, request.Rate, request.Units)
		if err != nil {
			return nil, err
		}
		entries[i] = e
		allAllowed = allAllowed && decisions[i].allowed
	}

	results := make([]Result, len(requests))
	for i, request := range requests {
		d := decisions[i]
		level := d.levelBefore
		spent := int64(0)
		if allAllowed {
			entries[i].apply(d, l.podID)
			level = d.levelAfter
			spent = request.Units
		}
		results[i] = Result{
			Allowed:   d.allowed,
			Remaining: request.Rate.remaining(d.levelBefore, spent),
			ResetAt:   time.Unix(0, nowNs+request.Rate.nextUnitAfterNs(level)),
		}
	}
	return results, nil
}

// Close is safe to call concurrently and more than once; only the first call
// flushes. The caller must not hold a shard lock.
func (l *limiter) Close() error {
	if l.closed.Swap(true) {
		return nil
	}
	if l.stopLoops != nil {
		l.stopLoops()
		l.loopsDone.Wait()
	}
	ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
	defer cancel()
	return l.flush(ctx)
}

// flushUntilStopped flushes on every tick until `ctx` is canceled. Only New
// starts it, in one goroutine.
func (l *limiter) flushUntilStopped(ctx context.Context, ticker clock.Ticker) {
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			flushCtx, cancel := context.WithTimeout(ctx, flushTimeout)
			err := l.flush(flushCtx)
			cancel()
			if err != nil {
				logger.Warn("unable to flush rate limit rows", "error", err.Error())
			}
		}
	}
}

// flush writes every uncommitted row to the store and removes entries that
// `entry.evictable` allows. A failed or lost write leaves its rows
// uncommitted, so the next flush sends their latest sums, which include the
// lost units. A row gets at most one write per flush, so a pod writes one
// row per active limit per flush, two when a period ends between flushes,
// however many requests it allowed.
//
// It is safe to call concurrently: `flushMu` runs one flush at a time, and it
// takes each shard lock itself. The caller must not hold a shard lock.
func (l *limiter) flush(ctx context.Context) error {
	l.flushMu.Lock()
	defer l.flushMu.Unlock()
	nowNs := l.clock.Now().UnixNano()

	var writes []StoredRow
	for i := range l.shards {
		s := &l.shards[i]
		s.mu.Lock()
		for fingerprint, e := range s.entries {
			if e.evictable(nowNs) {
				delete(s.entries, fingerprint)
				continue
			}
			for key, r := range e.uncommittedRows() {
				writes = append(writes, StoredRow{
					Fingerprint:   fingerprint,
					PeriodStartNs: key.periodStartNs,
					PodID:         l.podID,
					ScaledSum:     r.scaledSum,
					DeleteAfterNs: deleteAfterNs(key.periodStartNs, e.periodNs, e.duration, r.scaledSum),
				})
			}
		}
		s.mu.Unlock()
	}
	// Map order is random; sorting keeps simulation runs repeatable.
	slices.SortFunc(writes, func(left, right StoredRow) int {
		return cmp.Or(bytes.Compare(left.Fingerprint[:], right.Fingerprint[:]), cmp.Compare(left.PeriodStartNs, right.PeriodStartNs))
	})

	var errs []error
	for chunk := range slices.Chunk(writes, maxRowsPerWrite) {
		err := l.store.WriteRows(ctx, chunk)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, written := range chunk {
			s := &l.shards[shardIndex(written.Fingerprint)]
			s.mu.Lock()
			// Only flush removes entries, and it never removes one with
			// uncommitted rows, so the entry still exists.
			e := s.entries[written.Fingerprint]
			key := rowKey{periodStartNs: written.PeriodStartNs, podID: l.podID}
			committed := e.rows[key]
			committed.committedSum = max(committed.committedSum, written.ScaledSum)
			e.rows[key] = committed
			s.mu.Unlock()
		}
	}
	return errors.Join(errs...)
}

// refreshRows makes sure that the peer rows of each request are less than
// `rowsStaleFor` old, or that a read for them failed within that time. A
// failed read leaves the entry with the rows it already has, so the pod
// limits on what it knows until MySQL is back.
//
// It is safe to call concurrently. It takes the shard locks itself and does
// not hold them while it reads or waits, so the caller must not hold a shard
// lock.
func (l *limiter) refreshRows(ctx context.Context, requests []Request) {
	nowNs := l.clock.Now().UnixNano()
	var readNow [][32]byte
	var waitFor []chan struct{}
	for _, request := range requests {
		s := &l.shards[shardIndex(request.Fingerprint)]
		s.mu.Lock()
		e := l.entryOf(request.Fingerprint, request.Rate)
		ageNs := nowNs - e.rowsReadAtNs
		mustWait := ageNs >= rowsStaleFor.Nanoseconds()
		switch {
		case ageNs < rowsFreshFor.Nanoseconds():
		case e.readDone != nil:
			if mustWait {
				waitFor = append(waitFor, e.readDone)
			}
		case mustWait:
			e.readDone = make(chan struct{})
			readNow = append(readNow, request.Fingerprint)
		default:
			e.readDone = make(chan struct{})
			fingerprint := request.Fingerprint
			l.runInBackground(func() { l.readRows(context.Background(), fingerprint, readTimeout) })
		}
		s.mu.Unlock()
	}
	// Other requests may wait for these reads, so a canceled request must
	// not cancel them. They run at the same time; the last one runs here.
	for i, fingerprint := range readNow {
		if i == len(readNow)-1 {
			l.readRows(context.WithoutCancel(ctx), fingerprint, readWaitTimeout)
			continue
		}
		done := make(chan struct{})
		waitFor = append(waitFor, done)
		go func() {
			defer close(done)
			l.readRows(context.WithoutCancel(ctx), fingerprint, readWaitTimeout)
		}()
	}
	for _, done := range waitFor {
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
	}
}

// errReadsSaturated means that `readsInFlightMax` reads were running.
var errReadsSaturated = errors.New("too many rate limit reads are running")

// readWithinSlots reads the rows of one fingerprint within `timeout` if fewer
// than `readsInFlightMax` reads run, and returns `errReadsSaturated`
// otherwise. It is safe to call concurrently. The caller must not hold a
// shard lock, because it waits on the store.
func (l *limiter) readWithinSlots(ctx context.Context, fingerprint [32]byte, timeout time.Duration) ([]StoredRow, error) {
	select {
	case l.readSlots <- struct{}{}:
	default:
		return nil, errReadsSaturated
	}
	defer func() { <-l.readSlots }()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return l.store.ReadRows(ctx, fingerprint, l.podID, maxPeriodsPerRead)
}

// readRows reads the rows of one fingerprint within `timeout` and merges them
// into its entry. The caller must set `readDone` on the entry first, so only
// one readRows runs per fingerprint. It reads without a lock and then takes
// the shard lock to merge, so the caller must not hold a shard lock.
func (l *limiter) readRows(ctx context.Context, fingerprint [32]byte, timeout time.Duration) {
	startedAtNs := l.clock.Now().UnixNano()
	stored, err := l.readWithinSlots(ctx, fingerprint, timeout)
	periods := make(map[int64]struct{}, len(stored))
	for _, r := range stored {
		periods[r.PeriodStartNs] = struct{}{}
	}
	periodsPerRead.Observe(float64(len(periods)))
	result := "ok"
	switch {
	case errors.Is(err, errReadsSaturated):
		result = "saturated"
	case err != nil:
		result = "error"
	}

	s := &l.shards[shardIndex(fingerprint)]
	s.mu.Lock()
	defer s.mu.Unlock()
	// Flush keeps an entry while `readDone` is set, so it still exists.
	e := s.entries[fingerprint]
	if err == nil {
		err = e.replaceRows(stored, l.podID, l.clock.Now().UnixNano())
		if err != nil {
			result = "invalid_rows"
		}
	}
	// A failed read counts as a read, so a down database gets at most one
	// waiting read per limit every `rowsStaleFor`.
	e.rowsReadAtNs = startedAtNs
	e.rowsReadOK = err == nil
	close(e.readDone)
	e.readDone = nil
	readsTotal.WithLabelValues(result).Inc()
}

// validate checks the requests and that the limiter is open. It is safe to
// call concurrently. It takes shard locks itself, so the caller must not hold
// a shard lock.
func (l *limiter) validate(requests []Request) error {
	if l.closed.Load() {
		return fault.New("the rate limiter is closed")
	}
	fingerprints := make(map[[32]byte]struct{}, len(requests))
	for _, request := range requests {
		_, duplicate := fingerprints[request.Fingerprint]
		err := assert.All(
			request.Rate.Validate(),
			assert.GreaterOrEqual(request.Units, 0, "units must not be negative"),
			assert.False(duplicate, "fingerprints must be distinct"),
			l.checkDuration(request),
		)
		if err != nil {
			return err
		}
		fingerprints[request.Fingerprint] = struct{}{}
	}
	return nil
}

// checkDuration returns an error if this pod knows the fingerprint of a
// request with another duration, which means that the fingerprint lacks the
// duration. It is safe to call concurrently. It takes the shard lock
// itself, so the caller must not hold a shard lock.
func (l *limiter) checkDuration(request Request) error {
	s := &l.shards[shardIndex(request.Fingerprint)]
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[request.Fingerprint]
	if !ok {
		return nil
	}
	return assert.Equal(e.duration, request.Rate.Duration, "the fingerprint must include the duration")
}

// entryOf returns the entry of a fingerprint and creates it if needed. The
// caller must hold the shard lock.
func (l *limiter) entryOf(fingerprint [32]byte, rate Rate) *entry {
	entries := l.shards[shardIndex(fingerprint)].entries
	e, ok := entries[fingerprint]
	if !ok {
		e = newEntry(rate.Duration)
		entries[fingerprint] = e
	}
	return e
}

// randomInt64N returns a random number in [0, n). It is safe to call
// concurrently: it takes `randomMu`. The caller may hold a shard lock.
func (l *limiter) randomInt64N(n int64) int64 {
	l.randomMu.Lock()
	defer l.randomMu.Unlock()
	return l.random.Int64N(n)
}

// shardIndex returns the shard of a fingerprint. It is safe to call
// concurrently.
func shardIndex(fingerprint [32]byte) int {
	return int(fingerprint[0]) % shardCount
}

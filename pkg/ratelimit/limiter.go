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
	// reached each other yet. The over-admission is paid back afterwards.
	AllowMany(context.Context, []Request) ([]Result, error)

	// Close stops the background work and writes every session that MySQL has
	// not caught up with. It returns an error if that write fails. Allow and
	// AllowMany return an error after Close.
	Close() error
}

// Config holds the dependencies of a [Limiter].
type Config struct {
	// Clock gives the time of every decision and drives session lifetimes.
	// Tests and simulations pass a `clock.TestClock`. Required.
	Clock clock.Clock

	// Store holds the sessions that all pods share. Required.
	Store Store

	// Random generates session IDs and session lifetimes. Tests and
	// simulations pass a seeded one to make runs repeatable. Optional; the
	// default is seeded from crypto/rand. Two pods must never share a seed:
	// equal session IDs would hide each other's units.
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
	// makes two limits share one bucket.
	//
	// Include `Rate.Duration` but not `Rate.Limit`. Two durations on the same
	// identifier are two limits. A changed limit must keep what was already
	// consumed.
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

	// ResetAt is when every unit is free again. It is never before the time
	// of the request.
	ResetAt time.Time
}

const (
	shardCount = 64

	sessionIDLength = 16

	// A session ends after a random lifetime in [min, max). The jitter keeps
	// pods from starting new sessions all at once. Rows are compacted 30
	// minutes after their anchor, so a session must end well before that.
	minSessionLifetime = 12 * time.Minute
	maxSessionLifetime = 15 * time.Minute

	// compactAfter is how long after its anchor a row may be compacted. A
	// pod writes a session within this time or drops its unwritten units.
	compactAfter = 30 * time.Minute

	flushInterval       = time.Second
	flushTimeout        = 5 * time.Second
	maxSessionsPerWrite = 10_000

	// Peer rows younger than rowsFreshFor are used as they are. Older rows
	// are used while a background read refreshes them. A request waits for
	// a read only when the rows are rowsStaleFor or older.
	rowsFreshFor   = time.Second
	rowsStaleFor   = 5 * time.Second
	readTimeout    = time.Second
	maxRowsPerRead = 1_000
)

const sessionIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type limiter struct {
	clock clock.Clock
	store Store

	// randomMu guards random, which is not safe for concurrent use.
	randomMu sync.Mutex
	random   *mathrand.Rand

	shards [shardCount]shard
	closed atomic.Bool

	// flushMu lets one flush run at a time, so only the flush that collected
	// a session marks it as written.
	flushMu sync.Mutex

	// runInBackground runs background reads. It is `go f()`; the
	// simulation replaces it to choose when background reads run.
	runInBackground func(f func())

	// stopFlushLoop and flushLoopDone are nil if no flush loop runs.
	stopFlushLoop chan struct{}
	flushLoopDone chan struct{}
}

// shard holds the entries of the fingerprints whose first byte maps to it.
// mu guards entries and every entry in it.
type shard struct {
	mu      sync.Mutex
	entries map[[32]byte]*entry
}

// New returns a [Limiter] that flushes every second. It is safe to call
// concurrently.
func New(config Config) (Limiter, error) {
	l, err := newLimiter(config)
	if err != nil {
		return nil, err
	}
	l.stopFlushLoop = make(chan struct{})
	l.flushLoopDone = make(chan struct{})
	go l.flushUntilStopped(l.clock.NewTicker(flushInterval))
	return l, nil
}

// newLimiter returns a limiter without the flush loop. Tests and the
// simulation use it and call flush themselves, so the order of events
// follows their seed and not the Go scheduler. It is safe to call
// concurrently.
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
			return nil, fault.Wrap(err, fault.Internal("unable to seed the session ID generator"))
		}
		random = mathrand.New(mathrand.NewPCG(binary.LittleEndian.Uint64(seed[:8]), binary.LittleEndian.Uint64(seed[8:])))
	}
	l := &limiter{
		clock:    config.Clock,
		store:    config.Store,
		randomMu: sync.Mutex{},
		random:   random,
		shards:   [shardCount]shard{},
		closed:   atomic.Bool{},
		flushMu:  sync.Mutex{},
		runInBackground: func(f func()) {
			go f()
		},
		stopFlushLoop: nil,
		flushLoopDone: nil,
	}
	for i := range l.shards {
		l.shards[i].entries = make(map[[32]byte]*entry)
	}
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
		e := l.entryOf(request.Fingerprint)
		if e.liveID != "" && nowNs >= e.liveSessionEndsAtNs {
			e.endLiveSession()
		}
		decisions[i], err = e.check(nowNs, request.Rate, request.Units)
		if err != nil {
			return nil, err
		}
		entries[i] = e
		allAllowed = allAllowed && decisions[i].allowed
	}

	results := make([]Result, len(requests))
	for i, request := range requests {
		e := entries[i]
		if allAllowed {
			err = l.apply(e, decisions[i], nowNs)
			if err != nil {
				return nil, err
			}
		}
		results[i] = Result{
			Allowed:   decisions[i].allowed,
			Remaining: request.Rate.remaining(nowNs, e.resetTimes.resetAtNs),
			ResetAt:   time.Unix(0, max(e.resetTimes.resetAtNs, nowNs)),
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
	if l.stopFlushLoop != nil {
		close(l.stopFlushLoop)
		<-l.flushLoopDone
	}
	ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
	defer cancel()
	return l.flush(ctx)
}

// flushUntilStopped flushes on every tick until Close. Only New starts it, in
// one goroutine.
func (l *limiter) flushUntilStopped(ticker clock.Ticker) {
	defer close(l.flushLoopDone)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopFlushLoop:
			return
		case <-ticker.C():
			ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
			err := l.flush(ctx)
			cancel()
			if err != nil {
				logger.Warn("unable to flush rate limit sessions", "error", err.Error())
			}
		}
	}
}

// flush writes every unwritten session to the store and removes entries that
// are idle and fully written. A failed write leaves its sessions unwritten,
// so the next flush retries them with their latest values.
//
// It is safe to call concurrently: `flushMu` runs one flush at a time, and it
// takes each shard lock itself. The caller must not hold a shard lock.
func (l *limiter) flush(ctx context.Context) error {
	l.flushMu.Lock()
	defer l.flushMu.Unlock()
	nowNs := l.clock.Now().UnixNano()

	var writes []StoredSession
	for i := range l.shards {
		s := &l.shards[i]
		s.mu.Lock()
		for fingerprint, e := range s.entries {
			unwritten := e.unwrittenSessions(nowNs)
			// An idle entry decides like a new one, so removing it changes
			// no decision. A running read merges into the entry when it
			// finishes, so the entry stays until then.
			if len(unwritten) == 0 && e.resetTimes.resetAtNs < nowNs && e.readDone == nil {
				delete(s.entries, fingerprint)
				continue
			}
			for _, session := range unwritten {
				writes = append(writes, StoredSession{
					Fingerprint:    fingerprint,
					SessionID:      session.id,
					AnchorAtNs:     session.anchorAtNs,
					ConsumedNs:     session.consumedNs,
					CompactAfterMs: (session.anchorAtNs + compactAfter.Nanoseconds()) / time.Millisecond.Nanoseconds(),
				})
			}
		}
		s.mu.Unlock()
	}
	// Map order is random; sorting keeps simulation runs repeatable.
	slices.SortFunc(writes, func(left, right StoredSession) int {
		return cmp.Or(bytes.Compare(left.Fingerprint[:], right.Fingerprint[:]), cmp.Compare(left.SessionID, right.SessionID))
	})

	var errs []error
	for chunk := range slices.Chunk(writes, maxSessionsPerWrite) {
		err := l.store.WriteSessions(ctx, chunk)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, written := range chunk {
			s := &l.shards[shardIndex(written.Fingerprint)]
			s.mu.Lock()
			// Only flush removes entries, and it never removes one with
			// unwritten sessions, so the entry still exists.
			s.entries[written.Fingerprint].markWritten(written.SessionID, written.ConsumedNs)
			s.mu.Unlock()
		}
	}
	return errors.Join(errs...)
}

// refreshRows makes sure that the peer rows of each request are less than
// `rowsStaleFor` old, or that a read for them failed within that time. A
// failed read leaves the entry with the sessions it already has, so the pod
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
		e := l.entryOf(request.Fingerprint)
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
			l.runInBackground(func() { l.readRows(context.Background(), fingerprint) })
		}
		s.mu.Unlock()
	}
	// Other requests may wait for these reads, so a canceled request must
	// not cancel them.
	for _, fingerprint := range readNow {
		l.readRows(context.WithoutCancel(ctx), fingerprint)
	}
	for _, done := range waitFor {
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
	}
}

// readRows reads the peer rows of one fingerprint and merges them into its
// entry. The caller set `readDone` on the entry, so only one readRows runs
// per fingerprint. It reads without a lock and then takes the shard lock to
// merge, so the caller must not hold a shard lock.
func (l *limiter) readRows(ctx context.Context, fingerprint [32]byte) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	startedAtNs := l.clock.Now().UnixNano()
	stored, err := l.store.ReadSessions(ctx, fingerprint, maxRowsPerRead)
	rowsPerRead.Observe(float64(len(stored)))
	result := "ok"
	switch {
	case err != nil:
		result = "error"
	case len(stored) >= maxRowsPerRead:
		// Older rows may be missing, and a missing row can open the limit
		// too early, so the page is ignored.
		result = "row_limit"
		err = fault.New("the read reached the row limit")
	}

	s := &l.shards[shardIndex(fingerprint)]
	s.mu.Lock()
	defer s.mu.Unlock()
	// Flush keeps an entry while `readDone` is set, so it still exists.
	e := s.entries[fingerprint]
	if err == nil {
		rows := make([]session, len(stored))
		for i, row := range stored {
			rows[i] = session{id: row.SessionID, anchorAtNs: row.AnchorAtNs, consumedNs: row.ConsumedNs}
		}
		err = e.mergeRows(rows)
		if err != nil {
			result = "invalid_rows"
		}
	}
	// A failed read counts as a read, so a down database gets at most one
	// waiting read per limit every `rowsStaleFor`.
	e.rowsReadAtNs = startedAtNs
	close(e.readDone)
	e.readDone = nil
	readsTotal.WithLabelValues(result).Inc()
}

// validate checks the requests and that the limiter is open. It reads only
// its arguments and `closed`, so it is safe to call concurrently.
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
		)
		if err != nil {
			return err
		}
		fingerprints[request.Fingerprint] = struct{}{}
	}
	return nil
}

// entryOf returns the entry of a fingerprint and creates it if needed. The
// caller holds the shard lock.
func (l *limiter) entryOf(fingerprint [32]byte) *entry {
	entries := l.shards[shardIndex(fingerprint)].entries
	e, ok := entries[fingerprint]
	if !ok {
		e = new(entry)
		entries[fingerprint] = e
	}
	return e
}

// apply writes an allowed decision into the entry. A new session gets a fresh
// ID and a random lifetime. The caller holds the shard lock.
func (l *limiter) apply(e *entry, d decision, nowNs int64) error {
	if d.createNewSessionAtNs == 0 {
		return e.apply(d, "")
	}
	id, lifetime := l.newSession()
	err := e.apply(d, id)
	if err != nil {
		return err
	}
	e.liveSessionEndsAtNs = nowNs + lifetime.Nanoseconds()
	return nil
}

// newSession returns a random session ID and lifetime. It is safe to call
// concurrently: it takes `randomMu`. The caller may hold a shard lock.
func (l *limiter) newSession() (string, time.Duration) {
	l.randomMu.Lock()
	defer l.randomMu.Unlock()
	id := make([]byte, sessionIDLength)
	for i := range id {
		id[i] = sessionIDAlphabet[l.random.IntN(len(sessionIDAlphabet))]
	}
	lifetime := minSessionLifetime + time.Duration(l.random.Int64N((maxSessionLifetime - minSessionLifetime).Nanoseconds()))
	return string(id), lifetime
}

// shardIndex returns the shard of a fingerprint. It is safe to call
// concurrently.
func shardIndex(fingerprint [32]byte) int {
	return int(fingerprint[0]) % shardCount
}

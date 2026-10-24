package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
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
	// not caught up with. It returns an error if that write fails. Calls to
	// the Limiter after Close return an error.
	Close() error
}

// Config holds the dependencies of a [Limiter].
type Config struct {
	// Clock gives the time of every decision and drives session lifetimes.
	// Tests and simulations pass a `clock.TestClock`. Required.
	Clock clock.Clock

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
)

const sessionIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type limiter struct {
	clock clock.Clock

	// randomMu guards random, which is not safe for concurrent use.
	randomMu sync.Mutex
	random   *mathrand.Rand

	shards [shardCount]shard
	closed atomic.Bool
}

type shard struct {
	mu      sync.Mutex
	entries map[[32]byte]*entry
}

// New returns a [Limiter].
func New(config Config) (Limiter, error) {
	err := assert.NotNil(config.Clock, "clock is required")
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
		randomMu: sync.Mutex{},
		random:   random,
		shards:   [shardCount]shard{},
		closed:   atomic.Bool{},
	}
	for i := range l.shards {
		l.shards[i].entries = make(map[[32]byte]*entry)
	}
	return l, nil
}

func (l *limiter) Allow(ctx context.Context, request Request) (Result, error) {
	results, err := l.AllowMany(ctx, []Request{request})
	if err != nil {
		return Result{}, err
	}
	return results[0], nil
}

func (l *limiter) AllowMany(_ context.Context, requests []Request) ([]Result, error) {
	err := l.validate(requests)
	if err != nil {
		return nil, err
	}
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

func (l *limiter) Close() error {
	l.closed.Store(true)
	return nil
}

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

// newSession returns a random session ID and lifetime.
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

func shardIndex(fingerprint [32]byte) int {
	return int(fingerprint[0]) % shardCount
}

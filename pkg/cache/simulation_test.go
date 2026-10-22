package cache_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/sim"
)

// refreshQueue is a cache.Scheduler that holds refreshes until an event runs
// one, so the seed decides when each refresh happens.
type refreshQueue struct {
	refreshes []func()
	limit     int
}

func (q *refreshQueue) Schedule(refresh func()) bool {
	if len(q.refreshes) >= q.limit {
		return false
	}
	q.refreshes = append(q.refreshes, refresh)
	return true
}

// modelEntry is what the cache must return for a key while it still holds it.
type modelEntry struct {
	value uint64
	hit   cache.CacheHit
	fresh time.Time
	stale time.Time
}

type state struct {
	clk     *clock.TestClock
	cache   cache.Cache[uint64, uint64]
	queue   *refreshQueue
	model   map[uint64]modelEntry
	fresh   time.Duration
	stale   time.Duration
	maxSize int
	keys    uint64
	version uint64
	trace   hash.Hash64

	// calls counts origin calls, so an event can tell whether the cache
	// answered from memory or asked the origin. outcome and nextOp describe
	// the latest origin call; op returns nextOp the way a caller's op maps
	// the error it just got.
	calls   int
	outcome outcome
	nextOp  cache.Op
	// inFallback is true while swrWithFallbackEvent waits for SWRWithFallback.
	// SWRWithFallback and its background refresh write different things for
	// the same answer, so the origin needs to know which one called it.
	inFallback bool
}

// outcome is one answer a simulated origin can give.
type outcome int

const (
	// found returns a new value, and the caller's op writes it.
	found outcome = iota
	// notFound returns sql.ErrNoRows, and the caller's op writes Null.
	notFound
	// empty returns no value and no error, and the caller's op writes Null.
	empty
	// skipped returns a value, but the caller's op chooses not to cache it.
	skipped
	// failed returns a transient error, and the caller's op writes nothing.
	failed
)

var errOriginDown = errors.New("origin unavailable")

func pickOutcome(rng *rand.Rand) outcome {
	switch n := rng.IntN(10); {
	case n < 6:
		return found
	case n == 6:
		return notFound
	case n == 7:
		return empty
	case n == 8:
		return skipped
	default:
		return failed
	}
}

func (s *state) op(error) cache.Op { return s.nextOp }

// answer records a new origin call and picks its outcome.
func (s *state) answer(rng *rand.Rand) outcome {
	s.calls++
	s.version++
	s.outcome = pickOutcome(rng)
	s.nextOp = map[outcome]cache.Op{found: cache.WriteValue, notFound: cache.WriteNull, empty: cache.WriteNull, skipped: cache.Noop, failed: cache.Noop}[s.outcome]
	s.record("origin %d %d", s.version, s.outcome)
	return s.outcome
}

func (s *state) key(rng *rand.Rand) uint64 {
	return rng.Uint64N(s.keys)
}

func (s *state) record(format string, args ...any) {
	fmt.Fprintf(s.trace, format+"\n", args...)
}

func (s *state) remember(key, value uint64, hit cache.CacheHit) {
	now := s.clk.Now()
	s.model[key] = modelEntry{value: value, hit: hit, fresh: now.Add(s.fresh), stale: now.Add(s.stale)}
}

// anyStale reports whether one of keys is past its fresh time but still
// servable, the only state in which a read may ask for a refresh.
func (s *state) anyStale(keys ...uint64) bool {
	now := s.clk.Now()
	for _, key := range keys {
		if e, ok := s.model[key]; ok && !now.Before(e.fresh) && now.Before(e.stale) {
			return true
		}
	}
	return false
}

// checkNoRefresh fails when a read of keys that were all fresh or absent
// queued a refresh. queued is the queue length before the read.
func (s *state) checkNoRefresh(op string, stale bool, queued int, keys ...uint64) error {
	if !stale && len(s.queue.refreshes) > queued {
		return fmt.Errorf("%s(%v) scheduled a refresh although no key was stale", op, keys)
	}
	return nil
}

// origin answers SWR and its background refresh for key. The model learns
// every write at the same instant the cache makes it. The closure keeps rng,
// so a refresh that runs later still draws from the seeded sequence.
func (s *state) origin(rng *rand.Rand, key uint64) func(context.Context) (uint64, error) {
	return func(context.Context) (uint64, error) {
		switch s.answer(rng) {
		case found:
			s.remember(key, s.version, cache.Hit)
			return s.version, nil
		case notFound:
			s.remember(key, 0, cache.Null)
			return 0, sql.ErrNoRows
		case empty:
			s.remember(key, 0, cache.Null)
			return 0, nil
		case skipped:
			return s.version, nil
		case failed:
		}
		return 0, errOriginDown
	}
}

// originMany answers SWRMany and its background refresh. When it finds
// values, every third key is not found, so the cache writes Null for it, and
// half of the answers also carry a key nobody asked for, which the cache
// writes too.
func (s *state) originMany(rng *rand.Rand) func(context.Context, []uint64) (map[uint64]uint64, error) {
	return func(_ context.Context, keys []uint64) (map[uint64]uint64, error) {
		switch s.answer(rng) {
		case notFound:
			for _, key := range keys {
				s.remember(key, 0, cache.Null)
			}
			return nil, sql.ErrNoRows
		case empty:
			for _, key := range keys {
				s.remember(key, 0, cache.Null)
			}
			return nil, nil
		case skipped:
			return map[uint64]uint64{keys[0]: s.version}, nil
		case failed:
			return nil, errOriginDown
		case found:
		}
		values := map[uint64]uint64{}
		for _, key := range keys {
			s.version++
			if s.version%3 == 0 {
				s.remember(key, 0, cache.Null)
				continue
			}
			values[key] = s.version
			s.remember(key, s.version, cache.Hit)
		}
		if extra := (keys[0] + 1) % s.keys; !slices.Contains(keys, extra) && rng.IntN(2) == 0 {
			s.version++
			values[extra] = s.version
			s.remember(extra, s.version, cache.Hit)
		}
		return values, nil
	}
}

// originFallback answers SWRWithFallback and its background refresh with a
// canonical key picked from candidates. SWRWithFallback returns a not-found
// error without writing, while its background refresh writes Null.
func (s *state) originFallback(rng *rand.Rand, candidates []uint64) func(context.Context) (uint64, uint64, error) {
	return func(context.Context) (uint64, uint64, error) {
		o := s.answer(rng)
		canonical := candidates[rng.IntN(len(candidates))]
		switch o {
		case found:
			s.remember(canonical, s.version, cache.Hit)
			return s.version, canonical, nil
		case notFound:
			if !s.inFallback {
				s.remember(canonical, 0, cache.Null)
			}
			return 0, canonical, sql.ErrNoRows
		case empty:
			s.remember(canonical, 0, cache.Null)
			return 0, canonical, nil
		case skipped:
			return s.version, canonical, nil
		case failed:
		}
		return 0, canonical, errOriginDown
	}
}

// fromOrigin returns what a read that just called the origin must return,
// given the origin's outcome.
func (s *state) fromOrigin() (uint64, cache.CacheHit, error) {
	switch s.outcome {
	case found:
		return s.version, cache.Hit, nil
	case notFound:
		return 0, cache.Miss, sql.ErrNoRows
	case empty:
		return 0, cache.Null, nil
	case skipped:
		return s.version, cache.Null, nil
	case failed:
	}
	return 0, cache.Miss, errOriginDown
}

func (s *state) checkOrigin(op string, value uint64, hit cache.CacheHit, err error) error {
	wantValue, wantHit, wantErr := s.fromOrigin()
	if value != wantValue || hit != wantHit || !errors.Is(err, wantErr) {
		return fmt.Errorf("%s returned %d/%v/%v after origin outcome %d, want %d/%v/%v", op, value, hit, err, s.outcome, wantValue, wantHit, wantErr)
	}
	return nil
}

// check fails when the cache returned something other than the model's
// current entry. A miss is always allowed for a live key, because S3-FIFO may
// have evicted it, but an expired or removed key must miss.
func (s *state) check(op string, key, value uint64, hit cache.CacheHit) error {
	s.record("%s %d %d %d", op, key, value, hit)
	want, ok := s.model[key]
	live := ok && s.clk.Now().Before(want.stale)
	if hit == cache.Miss {
		return nil
	}
	if !live {
		return fmt.Errorf("%s(%d) returned %d/%v for a key that is expired or removed", op, key, value, hit)
	}
	if hit != want.hit || value != want.value {
		return fmt.Errorf("%s(%d) returned %d/%v, want %d/%v", op, key, value, hit, want.value, want.hit)
	}
	return nil
}

type setEvent struct{}

func (setEvent) Name() string { return "set" }
func (setEvent) Run(rng *rand.Rand, s *state) error {
	key, value := s.key(rng), rng.Uint64()
	s.cache.Set(context.Background(), key, value)
	s.remember(key, value, cache.Hit)
	s.record("set %d %d", key, value)
	return nil
}

type setNullEvent struct{}

func (setNullEvent) Name() string { return "setNull" }
func (setNullEvent) Run(rng *rand.Rand, s *state) error {
	key := s.key(rng)
	s.cache.SetNull(context.Background(), key)
	s.remember(key, 0, cache.Null)
	s.record("setNull %d", key)
	return nil
}

// setManyEvent writes one key, because SetMany writes a map in Go's random
// order and with more keys that order would decide which keys get evicted.
type setManyEvent struct{}

func (setManyEvent) Name() string { return "setMany" }
func (setManyEvent) Run(rng *rand.Rand, s *state) error {
	key, value := s.key(rng), rng.Uint64()
	s.cache.SetMany(context.Background(), map[uint64]uint64{key: value})
	s.remember(key, value, cache.Hit)
	s.record("setMany %d %d", key, value)
	return nil
}

type setNullManyEvent struct{}

func (setNullManyEvent) Name() string { return "setNullMany" }
func (setNullManyEvent) Run(rng *rand.Rand, s *state) error {
	keys := []uint64{s.key(rng), s.key(rng)}
	s.cache.SetNullMany(context.Background(), keys)
	for _, key := range keys {
		s.remember(key, 0, cache.Null)
	}
	s.record("setNullMany %v", keys)
	return nil
}

type getEvent struct{}

func (getEvent) Name() string { return "get" }
func (getEvent) Run(rng *rand.Rand, s *state) error {
	key := s.key(rng)
	value, hit := s.cache.Get(context.Background(), key)
	return s.check("get", key, value, hit)
}

type removeEvent struct{}

func (removeEvent) Name() string { return "remove" }
func (removeEvent) Run(rng *rand.Rand, s *state) error {
	key := s.key(rng)
	s.cache.Remove(context.Background(), key)
	delete(s.model, key)
	s.record("remove %d", key)
	return nil
}

type swrEvent struct{}

func (swrEvent) Name() string { return "swr" }
func (swrEvent) Run(rng *rand.Rand, s *state) error {
	key := s.key(rng)
	calls, queued, stale := s.calls, len(s.queue.refreshes), s.anyStale(key)
	value, hit, err := s.cache.SWR(context.Background(), key, s.origin(rng, key), s.op)
	s.record("swr %d %d %d %v", key, value, hit, err)
	if err := s.checkNoRefresh("swr", stale, queued, key); err != nil {
		return err
	}
	if s.calls > calls {
		return s.checkOrigin("swr", value, hit, err)
	}
	if err != nil || hit == cache.Miss {
		return fmt.Errorf("swr(%d) returned %v/%v without asking the origin", key, hit, err)
	}
	return s.check("swr", key, value, hit)
}

type swrManyEvent struct{}

func (swrManyEvent) Name() string { return "swrMany" }
func (swrManyEvent) Run(rng *rand.Rand, s *state) error {
	keys := []uint64{s.key(rng), s.key(rng), s.key(rng)}
	calls, queued, stale := s.calls, len(s.queue.refreshes), s.anyStale(keys...)
	values, hits, err := s.cache.SWRMany(context.Background(), keys, s.originMany(rng), s.op)
	if err := s.checkNoRefresh("swrMany", stale, queued, keys...); err != nil {
		return err
	}
	called := s.calls > calls
	var wantErr error
	if called && s.outcome == notFound {
		wantErr = sql.ErrNoRows
	}
	if called && s.outcome == failed {
		wantErr = errOriginDown
	}
	if !errors.Is(err, wantErr) {
		return fmt.Errorf("swrMany(%v) returned error %v, want %v (called=%t)", keys, err, wantErr, called)
	}
	for _, key := range keys {
		if hits[key] == cache.Miss && (!called || s.outcome == found || s.outcome == notFound || s.outcome == empty) {
			return fmt.Errorf("swrMany(%d) missed although the cache or the origin had an answer", key)
		}
		if err := s.check("swrMany", key, values[key], hits[key]); err != nil {
			return err
		}
	}
	return nil
}

type swrWithFallbackEvent struct{}

func (swrWithFallbackEvent) Name() string { return "swrWithFallback" }
func (swrWithFallbackEvent) Run(rng *rand.Rand, s *state) error {
	candidates := []uint64{s.key(rng), s.key(rng)}
	calls, queued, stale := s.calls, len(s.queue.refreshes), s.anyStale(candidates...)
	s.inFallback = true
	value, hit, err := s.cache.SWRWithFallback(context.Background(), candidates, s.originFallback(rng, candidates), s.op)
	s.inFallback = false
	s.record("swrWithFallback %v %d %d %v", candidates, value, hit, err)
	if err := s.checkNoRefresh("swrWithFallback", stale, queued, candidates...); err != nil {
		return err
	}
	if s.calls > calls {
		wantValue, wantHit, wantErr := s.fromOrigin()
		if s.outcome == skipped {
			wantHit = cache.Miss
		}
		if value != wantValue || hit != wantHit || !errors.Is(err, wantErr) {
			return fmt.Errorf("swrWithFallback returned %d/%v/%v after origin outcome %d, want %d/%v/%v", value, hit, err, s.outcome, wantValue, wantHit, wantErr)
		}
		return nil
	}
	if err != nil || hit == cache.Miss {
		return fmt.Errorf("swrWithFallback(%v) returned %v/%v without asking the origin", candidates, hit, err)
	}
	for _, key := range candidates {
		if s.check("swrWithFallback", key, value, hit) == nil {
			return nil
		}
	}
	return fmt.Errorf("swrWithFallback(%v) returned %d/%v, which no live candidate holds", candidates, value, hit)
}

type clearEvent struct{}

func (clearEvent) Name() string { return "clear" }
func (clearEvent) Run(_ *rand.Rand, s *state) error {
	s.cache.Clear(context.Background())
	clear(s.model)
	s.record("clear")
	return nil
}

type runRevalidationEvent struct{}

func (runRevalidationEvent) Name() string { return "runRevalidation" }
func (runRevalidationEvent) Run(rng *rand.Rand, s *state) error {
	if len(s.queue.refreshes) == 0 {
		return nil
	}
	i := rng.IntN(len(s.queue.refreshes))
	refresh := s.queue.refreshes[i]
	s.queue.refreshes = slices.Delete(s.queue.refreshes, i, i+1)
	refresh()
	s.record("revalidate %d", i)
	return nil
}

type advanceTimeEvent struct{}

func (advanceTimeEvent) Name() string { return "advanceTime" }
func (advanceTimeEvent) Run(rng *rand.Rand, s *state) error {
	limit := s.fresh
	if rng.IntN(10) == 0 {
		limit = s.stale
	}
	d := time.Duration(rng.Int64N(int64(limit)))
	s.clk.Tick(d)
	s.record("tick %d", d)
	return nil
}

// dumpEvent checks the whole cache at once: it never holds more than MaxSize
// entries, and every entry it holds matches the model.
type dumpEvent struct{}

func (dumpEvent) Name() string { return "dump" }
func (dumpEvent) Run(_ *rand.Rand, s *state) error {
	b, err := s.cache.Dump(context.Background())
	if err != nil {
		return err
	}
	var entries map[uint64]struct {
		Value uint64         `json:"value"`
		Hit   cache.CacheHit `json:"hit"`
	}
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	if len(entries) > s.maxSize {
		return fmt.Errorf("cache holds %d entries, MaxSize is %d", len(entries), s.maxSize)
	}
	for key, e := range entries {
		want, ok := s.model[key]
		if !ok || want.value != e.Value || want.hit != e.Hit {
			return fmt.Errorf("dump has %d=%d/%v, model has %v (present=%t)", key, e.Value, e.Hit, want, ok)
		}
	}
	s.record("dump %d", len(entries))
	return nil
}

// runSimulation runs one seeded simulation against a cache with a small
// MaxSize, so eviction happens often, and returns a hash of everything the
// cache returned.
func runSimulation(t *testing.T, seed sim.Seed) uint64 {
	t.Helper()
	simulation := sim.New[state](seed,
		sim.WithState(func(rng *rand.Rand) *state {
			clk := clock.NewTestClock(time.Unix(1_700_000_000, 0))
			fresh := time.Second + time.Duration(rng.Int64N(int64(time.Minute)))
			stale := fresh + time.Duration(rng.Int64N(int64(10*time.Minute)))
			maxSize := 1 + rng.IntN(32)
			queue := &refreshQueue{refreshes: nil, limit: 1 + rng.IntN(8)}
			c, err := cache.New(cache.Config[uint64, uint64]{
				Fresh:     fresh,
				Stale:     stale,
				MaxSize:   maxSize,
				Resource:  "simulation",
				Clock:     clk,
				Scheduler: queue,
			})
			require.NoError(t, err)
			t.Cleanup(c.Close)
			return &state{
				clk:        clk,
				cache:      c,
				queue:      queue,
				model:      map[uint64]modelEntry{},
				fresh:      fresh,
				stale:      stale,
				maxSize:    maxSize,
				keys:       1 + rng.Uint64N(64),
				version:    0,
				trace:      fnv.New64a(),
				calls:      0,
				outcome:    found,
				nextOp:     cache.WriteValue,
				inFallback: false,
			}
		}),
	)
	err := simulation.Run([]sim.Event[state]{
		setEvent{},
		setNullEvent{},
		setManyEvent{},
		setNullManyEvent{},
		getEvent{},
		removeEvent{},
		swrEvent{},
		swrManyEvent{},
		swrWithFallbackEvent{},
		runRevalidationEvent{},
		clearEvent{},
		advanceTimeEvent{},
		dumpEvent{},
	})
	require.NoError(t, err)
	require.Empty(t, simulation.Errors, "seed %s", seed)
	return simulation.State().trace.Sum64()
}

func TestSimulation(t *testing.T) {
	for i := range 5 {
		seed := sim.NewSeedFromInt(i)
		t.Run(seed.String(), func(t *testing.T) {
			first := runSimulation(t, seed)
			second := runSimulation(t, seed)
			require.Equal(t, first, second, "the same seed must produce the same run")
		})
	}
}

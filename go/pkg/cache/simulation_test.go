package cache_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/sim"
)

type state struct {
	keys  []uint64
	cache cache.Cache[uint64, uint64]
	clk   *clock.TestClock
}

type setEvent struct{}

func (e *setEvent) Name() string {
	return "set"
}

func (e *setEvent) Run(rng *rand.Rand, s *state) error {

	key := rng.Uint64()
	val := rng.Uint64()

	s.keys = append(s.keys, key)
	s.cache.Set(context.Background(), key, val)

	return nil
}

type getEvent struct{}

func (e *getEvent) Name() string {
	return "get"
}

func (e *getEvent) Run(rng *rand.Rand, s *state) error {

	stored := len(s.keys)
	index := rng.Intn(stored + 1)

	key := rng.Uint64()

	if index < stored {
		key = s.keys[index]
	}

	s.cache.Get(context.Background(), key)

	return nil
}

type removeEvent struct{}

func (e *removeEvent) Name() string {
	return "remove"
}

func (e *removeEvent) Run(rng *rand.Rand, s *state) error {

	stored := len(s.keys)
	index := rng.Intn(stored + 1)

	key := rng.Uint64()

	if index < stored {
		key = s.keys[index]
	}

	s.cache.Remove(context.Background(), key)

	return nil
}

type advanceTimeEvent struct {
	clk *clock.TestClock
}

func (e *advanceTimeEvent) Name() string {
	return "advanceTime"
}

func (e *advanceTimeEvent) Run(rng *rand.Rand, s *state) error {
	nanoseconds := rng.Int63n(60 * 60 * 1000 * 1000) // up to 1h

	e.clk.Tick(time.Duration(nanoseconds) * time.Nanosecond)

	return nil
}

func TestSimulation(t *testing.T) {

	for i := range 100 {
		seed := time.Now().UnixNano() + rand.Int63()
		t.Run(fmt.Sprintf("run=%d,seed=%d", i, seed), func(t *testing.T) {

			clk := clock.NewTestClock()

			s := sim.New[state](t,
				sim.WithSeed[state](seed),
				sim.WithSteps[state](1000000),
				sim.WithState(func(rng *rand.Rand) *state {
					minTime := 1738364400000 // 2025-01-01
					maxTime := 2527282800000 // 2050-01-01
					unixMilli := minTime + rng.Intn(maxTime-minTime)
					clk.Set(time.UnixMilli(int64(unixMilli)))

					fresh := time.Second + time.Duration(rng.Intn(60*60*1000))
					stale := fresh + time.Duration(rng.Intn(24*60*60*1000))

					c := cache.New[uint64, uint64](cache.Config[uint64, uint64]{
						Clock:    clk,
						Fresh:    fresh,
						Stale:    stale,
						Logger:   logging.NewNoop(),
						MaxSize:  rng.Intn(1_000_000),
						Resource: "test",
					})

					return &state{
						keys:  []uint64{},
						cache: c,
						clk:   clk,
					}
				}))

			s.Run([]sim.Event[state]{
				&setEvent{},
				&getEvent{},
				&removeEvent{},
				&advanceTimeEvent{clk},
			})

			require.Len(t, s.Errors, 0, "expected no errors")
		})
	}
}

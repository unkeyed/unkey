package cache_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
	if stored == 0 {
		// Skip if no keys stored yet
		return nil
	}

	index := rng.IntN(stored)
	key := s.keys[index]

	s.cache.Get(context.Background(), key)

	return nil
}

type removeEvent struct{}

func (e *removeEvent) Name() string {
	return "remove"
}

func (e *removeEvent) Run(rng *rand.Rand, s *state) error {
	stored := len(s.keys)
	if stored == 0 {
		// Skip if no keys stored yet
		return nil
	}

	index := rng.IntN(stored)
	key := s.keys[index]

	s.cache.Remove(context.Background(), key)

	return nil
}

type advanceTimeEvent struct{}

func (e *advanceTimeEvent) Name() string {
	return "advanceTime"
}

func (e *advanceTimeEvent) Run(rng *rand.Rand, s *state) error {
	// Random time advance between 0 and 1 hour
	nanoseconds := rng.Int64N(60 * 60 * 1000 * 1000 * 1000)
	s.clk.Tick(time.Duration(nanoseconds))
	return nil
}

func TestSimulation(t *testing.T) {
	sim.CheckEnabled(t)

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("run=%d", i), func(t *testing.T) {
			seed := sim.NewSeed()

			simulation := sim.New[state](seed,
				sim.WithState(func(rng *rand.Rand) *state {
					clk := clock.NewTestClock(time.Now())

					fresh := time.Second + time.Duration(rng.IntN(60*60*1000))*time.Millisecond
					stale := fresh + time.Duration(rng.IntN(24*60*60*1000))*time.Millisecond

					c, err := cache.New[uint64, uint64](cache.Config[uint64, uint64]{
						Clock:    clk,
						Fresh:    fresh,
						Stale:    stale,
						Logger:   logging.NewNoop(),
						MaxSize:  rng.IntN(1_000_000) + 1, // Ensure at least size 1
						Resource: "test",
					})
					require.NoError(t, err)
					return &state{
						keys:  []uint64{},
						cache: c,
						clk:   clk,
					}
				}),
			)

			// Define a validator that ensures we don't panic
			stateValidator := func(s *state) error {
				if s.cache == nil {
					return fmt.Errorf("cache should not be nil")
				}
				return nil
			}

			// Add the validator
			simulation = sim.WithValidator(stateValidator)(simulation)

			// Run the simulation with the events
			err := simulation.Run([]sim.Event[state]{
				&setEvent{},
				&getEvent{},
				&removeEvent{},
				&advanceTimeEvent{},
			})
			require.NoError(t, err)

			require.Empty(t, simulation.Errors, "simulation should complete without errors")
		})
	}
}

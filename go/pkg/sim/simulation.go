package sim

import (
	"fmt"
	"math/rand"
	"testing"
)

type Event[State any] interface {
	// Run executes the event logic.
	// State must allow parallel manipulation from multiple goroutines.
	Run(rng *rand.Rand, state *State) error

	// Name returns the name of the event for logging and debugging purposes.
	Name() string
}

type Simulation[State any] struct {
	t     *testing.T
	seed  int64
	rng   *rand.Rand
	steps int

	Errors []error

	state *State

	// Tracks how many configurations have been applied
	applied int
}

type apply[S any] func(*Simulation[S]) *Simulation[S]

func New[State any](t *testing.T, fns ...apply[State]) *Simulation[State] {

	seed := NewSeed()

	s := &Simulation[State]{
		t:    t,
		seed: seed,
		// nolint:gosec
		rng:     rand.New(rand.NewSource(seed)),
		steps:   1_000_000_000,
		state:   nil,
		Errors:  []error{},
		applied: 0,
	}

	for _, fn := range fns {
		s = fn(s)
		s.applied++
	}
	return s
}

func WithSeed[S any](seed int64) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		if s.applied > 0 {
			s.t.Fatalf("WithSeed called too late. If you need a custom seed, call WithSeed before any other configuration.")
		}

		s.seed = seed
		// nolint:gosec
		s.rng = rand.New(rand.NewSource(seed))
		return s
	}
}

func WithSteps[S any](steps int) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		s.steps = steps
		return s
	}
}

func WithState[S any](fn func(rng *rand.Rand) *S) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		s.state = fn(s.rng)
		return s
	}
}

// Run must not be called concurrently
func (s *Simulation[State]) Run(events []Event[State]) {
	s.t.Helper()

	if len(events) == 0 {
		return
	}

	fmt.Printf("Simulation [seed=%d], steps=%d\n", s.seed, s.steps)

	total := 0.0
	weights := make([]float64, len(events))

	for i := range weights {
		weights[i] = s.rng.Float64()
		total += weights[i]
	}

	for i := 0; i < s.steps; i++ {
		if i%(s.steps/10) == 0 {
			s.t.Logf("progress: %d%%\n", i*100/s.steps)
		}

		r := s.rng.Float64() * total

		// Find which bucket it falls into
		sum := 0.0
		var index int
		for j, w := range weights {
			sum += w
			if r <= sum {
				index = j
				break
			}
		}

		event := events[index]

		err := event.Run(s.rng, s.state)
		if err != nil {
			s.Errors = append(s.Errors, err)
		}
	}
}

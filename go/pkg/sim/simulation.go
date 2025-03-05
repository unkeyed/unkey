package sim

import (
	"fmt"
	"math/rand"
	"testing"
)

// Simulation manages a property-based testing simulation with a typed state.
// It provides a framework for running randomized sequences of events against
// a system state to discover edge cases and bugs that might not be found
// with traditional unit testing.
type Simulation[State any] struct {
	t       *testing.T // The test instance for reporting failures
	seed    int64      // Random seed for reproducibility
	rng     *rand.Rand // Random number generator
	steps   int        // Number of steps to run in the simulation
	Errors  []error    // Errors encountered during the simulation
	state   *State     // The current system state
	applied int        // Tracks how many configurations have been applied
}

// apply is a function type for configuring a simulation
type apply[S any] func(*Simulation[S]) *Simulation[S]

// New creates a new simulation for property-based testing.
// It initializes the simulation with the provided testing instance and
// applies any configuration functions provided.
//
// Example:
//
//	sim := sim.New[MyState](t,
//	    sim.WithSeed(12345),
//	    sim.WithSteps(1000000),
//	    sim.WithState(func(rng *rand.Rand) *MyState {
//	        // Initialize state with random values
//	        return &MyState{
//	            Counter: rng.Intn(100),
//	            Name:    fmt.Sprintf("test-%d", rng.Intn(1000)),
//	        }
//	    }),
//	)
func New[State any](t *testing.T, fns ...apply[State]) *Simulation[State] {
	// Get a random seed for reproducibility
	seed := NewSeed()

	// Initialize the simulation
	s := &Simulation[State]{
		t:       t,
		seed:    seed,
		rng:     rand.New(rand.NewSource(seed)), // nolint:gosec
		steps:   1_000_000_000,                  // Default to a very large number of steps
		state:   nil,
		Errors:  []error{},
		applied: 0,
	}

	// Apply configuration functions
	for _, fn := range fns {
		s = fn(s)
		s.applied++
	}
	return s
}

// WithSeed configures the simulation to use a specific random seed.
// This allows for reproducible test runs. If a failure occurs, the seed
// can be used to reproduce the exact sequence of events that led to the failure.
//
// Important: If you need a custom seed, call WithSeed before any other configuration.
func WithSeed[S any](seed int64) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		if s.applied > 0 {
			s.t.Fatalf("WithSeed called too late. If you need a custom seed, call WithSeed before any other configuration.")
		}

		s.seed = seed
		s.rng = rand.New(rand.NewSource(seed)) // nolint:gosec
		return s
	}
}

// WithSteps configures the number of steps to run in the simulation.
// Each step executes a randomly selected event from the provided events.
func WithSteps[S any](steps int) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		s.steps = steps
		return s
	}
}

// WithState configures the initial state for the simulation.
// The provided function receives a random number generator and should
// return a pointer to a newly initialized state with random values.
func WithState[S any](fn func(rng *rand.Rand) *S) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		s.state = fn(s.rng)
		return s
	}
}

// Run executes the simulation with the provided events.
// It runs the configured number of steps, selecting a random event at each step
// with weighted probability. Any errors are collected in the Errors slice.
//
// Example:
//
//	sim.Run([]sim.Event[MyState]{
//	    &AddEvent{},
//	    &RemoveEvent{},
//	    &UpdateEvent{},
//	})
func (s *Simulation[State]) Run(events []Event[State]) {
	s.t.Helper()

	if len(events) == 0 {
		return
	}

	fmt.Printf("Simulation [seed=%d], steps=%d\n", s.seed, s.steps)

	// Calculate event weights for random selection
	total := 0.0
	weights := make([]float64, len(events))

	for i := range weights {
		weights[i] = s.rng.Float64()
		total += weights[i]
	}

	// Run the simulation steps
	for i := 0; i < s.steps; i++ {
		// Log progress every 10%
		if i%(s.steps/10) == 0 {
			s.t.Logf("progress: %d%%\n", i*100/s.steps)
		}

		// Select a random event based on weights
		r := s.rng.Float64() * total
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

		// Run the selected event
		err := event.Run(s.rng, s.state)
		if err != nil {
			s.Errors = append(s.Errors, err)
		}
	}
}

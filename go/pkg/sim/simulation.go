package sim

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Validator[S any] func(*S) error

// Simulation manages a property-based testing simulation with a typed state.
// It provides a framework for running randomized sequences of events against
// a system state to discover edge cases and bugs that might not be found
// with traditional unit testing.
type Simulation[State any] struct {
	clock       *clock.TestClock
	seed        Seed       // Random seed for reproducibility
	rng         *rand.Rand // Random number generator
	ticks       int64      // Number of steps to run in the simulation
	timePerTick time.Duration
	Errors      []error            // Errors encountered during the simulation
	state       *State             // The current system state
	applied     int                // Tracks how many configurations have been applied
	validators  []Validator[State] // Add validators
	eventStats  map[string]int     // Track how many times each event is called
	logger      logging.Logger
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
func New[State any](seed Seed, fns ...apply[State]) *Simulation[State] {
	// Create a source for reproducible random number generation

	// nolint:gosec
	rng := rand.New(rand.NewChaCha8(seed))
	// Initialize the simulation
	s := &Simulation[State]{
		clock:       clock.NewTestClock(time.UnixMilli(2000000000000)),
		seed:        seed,
		rng:         rng,                   // Use our custom source
		ticks:       rng.Int64N(1_000_000), // Default to a reasonable number
		timePerTick: time.Duration(1+rng.IntN(10)) * time.Millisecond,
		state:       nil,
		Errors:      []error{},
		applied:     0,
		eventStats:  make(map[string]int), // Initialize event stats map
		logger:      logging.New(),
		validators:  []Validator[State]{},
	}

	// Apply configuration functions
	for _, fn := range fns {
		s = fn(s)
		s.applied++
	}
	return s
}
func WithValidator[S any](validator Validator[S]) apply[S] {
	return func(s *Simulation[S]) *Simulation[S] {
		s.validators = append(s.validators, validator)
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
func (s *Simulation[State]) Run(events []Event[State]) error {

	if len(events) == 0 {
		return fmt.Errorf("no events to run")
	}
	defer s.PrintEventStats()

	// Validate initial state
	for _, validator := range s.validators {
		if err := validator(s.state); err != nil {
			return fmt.Errorf("initial state validation failed: %w", err)
		}
	}

	s.logger.Info("Simulation",
		"seed", s.seed.String(),
		"ticks", s.ticks,
	)

	// Run the simulation steps
	for i := int64(0); i < s.ticks; i++ {
		s.clock.Tick(s.timePerTick)

		// Select a random event based on weights
		event := s.selectEvent(events)
		s.eventStats[event.Name()]++

		// Run the selected event
		err := event.Run(s.rng, s.state)
		if err != nil {
			s.Errors = append(s.Errors, err)
		}
	}
	return nil
}

func (s *Simulation[State]) State() *State {
	return s.state
}

// selectEvent chooses an event from the provided slice using a selection method
// determined by the random seed
func (s *Simulation[State]) selectEvent(events []Event[State]) Event[State] {
	// The first random choice determines the selection method
	selectionMethod := s.rng.IntN(3) // 0, 1, or 2

	switch selectionMethod {
	case 0: // Uniform selection
		return events[s.rng.IntN(len(events))]

	case 1: // Weighted selection with linear distribution
		weights := make([]float64, len(events))
		total := 0.0

		// Generate linearly decreasing weights
		for i := range weights {
			weights[i] = float64(len(events) - i)
			total += weights[i]
		}

		// Shuffle the weights to avoid bias toward specific events
		s.rng.Shuffle(len(weights), func(i, j int) {
			weights[i], weights[j] = weights[j], weights[i]
		})

		// Select based on weights
		r := s.rng.Float64() * total
		sum := 0.0
		for i, w := range weights {
			sum += w
			if r <= sum {
				return events[i]
			}
		}

	case 2: // Weighted selection with exponential bias
		weights := make([]float64, len(events))
		total := 0.0

		// Generate exponentially decreasing weights
		for i := range weights {
			weights[i] = 1.0 / float64(i+1) // 1, 1/2, 1/3, 1/4, ...
			total += weights[i]
		}

		// Shuffle the weights to avoid bias toward specific events
		s.rng.Shuffle(len(weights), func(i, j int) {
			weights[i], weights[j] = weights[j], weights[i]
		})

		// Select based on weights
		r := s.rng.Float64() * total
		sum := 0.0
		for i, w := range weights {
			sum += w
			if r <= sum {
				return events[i]
			}
		}
	}

	// Fallback
	return events[s.rng.IntN(len(events))]
}

func (s *Simulation[State]) PrintEventStats() {
	fmt.Println("\n--- Event Statistics ---")

	if len(s.eventStats) == 0 {
		fmt.Println("No events were executed.")
		return
	}

	// Calculate totals
	total := int64(0)
	for _, count := range s.eventStats {
		total += int64(count)
	}

	// Create a sorted list of event names for consistent output
	eventNames := make([]string, 0, len(s.eventStats))
	for name := range s.eventStats {
		eventNames = append(eventNames, name)
	}
	sort.Strings(eventNames)

	// Print stats for each event
	fmt.Printf("Total events executed: %d\n", total)
	fmt.Println("Distribution:")
	for _, name := range eventNames {
		count := s.eventStats[name]
		percentage := float64(count) / float64(total) * 100
		fmt.Printf("  %-20s: %6d (%6.2f%%)\n", name, count, percentage)
	}
	fmt.Println("------------------------")
}

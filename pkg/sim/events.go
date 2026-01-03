package sim

import (
	"errors"
	"math/rand/v2"
)

// Event defines the interface for simulation events that can be run against a state.
// Events are the building blocks of simulations, representing possible actions
// that can occur in the system being tested.
type Event[State any] interface {
	// Name returns a human-readable name for the event, used for logging and debugging.
	Name() string

	// Run executes the event logic against the given state using the provided
	// random number generator. It should modify the state according to the event's
	// logic and return any errors that occur during execution.
	Run(rng *rand.Rand, state *State) error
}

// Idle is a no-op event that does nothing.
// It's useful as a placeholder or to simulate periods of inactivity.
type Idle struct {
}

var _ Event[any] = (*Idle)(nil)

// Name returns the name of the idle event.
func (i Idle) Name() string {
	return "Idle"
}

// Run implements the Event interface but does nothing.
func (i Idle) Run(rng *rand.Rand, state *any) error {
	return nil
}

// Fail is an event that always fails with a specified error message.
// It's useful for testing error handling in simulations.
type Fail struct {
	// Message is the error message to return when the event runs.
	Message string
}

var _ Event[any] = (*Fail)(nil)

// Name returns the name of the fail event.
func (f Fail) Name() string {
	return "Fail"
}

// Run always returns an error with the specified message.
func (f Fail) Run(rng *rand.Rand, state *any) error {
	return errors.New(f.Message)
}

package sim_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/sim"
)

// Define a simple state for our test simulation
type TestState struct {
	Counter int
	Values  []string
}

// Add event: increments the counter
type AddEvent struct{}

func (e AddEvent) Name() string {
	return "Add"
}

func (e AddEvent) Run(rng *sim.Rand, state *TestState) error {
	state.Counter++
	return nil
}

// Subtract event: decrements the counter, fails if counter would become negative
type SubtractEvent struct{}

func (e SubtractEvent) Name() string {
	return "Subtract"
}

func (e SubtractEvent) Run(rng *sim.Rand, state *TestState) error {
	if state.Counter <= 0 {
		return fmt.Errorf("cannot subtract: counter would become negative")
	}
	state.Counter--
	return nil
}

// AddValue event: adds a random string to the values slice
type AddValueEvent struct{}

func (e AddValueEvent) Name() string {
	return "AddValue"
}

func (e AddValueEvent) Run(rng *sim.Rand, state *TestState) error {
	value := fmt.Sprintf("value-%d", rng.IntN(1000))
	state.Values = append(state.Values, value)
	return nil
}

// Test the basic simulation functionality
func TestBasicSimulation(t *testing.T) {
	sim.CheckEnabled(t)
	// Generate a new seed for the test
	seed := sim.NewSeed()

	// Initialize a simulation with a simple state
	simulation := sim.New[TestState](seed,
		sim.WithState(func(rng *sim.Rand) *TestState {
			return &TestState{
				Counter: 10, // Start with counter at 10
				Values:  []string{},
			}
		}),
	)

	// Define a validator to check the state constraints
	stateValidator := func(state *TestState) error {
		if state.Counter < 0 {
			return fmt.Errorf("invalid state: counter is negative (%d)", state.Counter)
		}
		return nil
	}

	// Add the validator to the simulation
	simulation = sim.WithValidator(stateValidator)(simulation)

	// Create a list of events for the simulation
	events := []sim.Event[TestState]{
		&AddEvent{},
		&SubtractEvent{},
		&AddValueEvent{},
	}

	// Run the simulation
	simulation.Run(events)

	// Verify final state
	require.NotNil(t, simulation.State())
	require.GreaterOrEqual(t, simulation.State().Counter, 0)

	// Print final state for debugging
	t.Logf("Final state: Counter=%d, Values=%d items",
		simulation.State().Counter, len(simulation.State().Values))
	require.Empty(t, simulation.Errors)
}

// Test the basic simulation functionality
func TestManySimulations(t *testing.T) {

	for i := range 100 {
		// Generate a new seed for the test
		seed := sim.NewSeedFromInt(i)

		// Initialize a simulation with a simple state
		simulation := sim.New[TestState](seed,
			sim.WithState(func(rng *sim.Rand) *TestState {
				return &TestState{
					Counter: 10, // Start with counter at 10
					Values:  []string{},
				}
			}),
		)

		// Define a validator to check the state constraints
		stateValidator := func(state *TestState) error {
			if state.Counter < 0 {
				return fmt.Errorf("invalid state: counter is negative (%d)", state.Counter)
			}
			return nil
		}

		// Add the validator to the simulation
		simulation = sim.WithValidator(stateValidator)(simulation)

		// Create a list of events for the simulation
		events := []sim.Event[TestState]{
			&AddEvent{},
			&SubtractEvent{},
			&AddValueEvent{},
		}

		// Run the simulation
		simulation.Run(events)

		// Verify final state
		require.GreaterOrEqual(t, simulation.State().Counter, 0)

		// Print final state for debugging
		t.Logf("Final state: Counter=%d, Values=%d items",
			simulation.State().Counter, len(simulation.State().Values))

	}
}

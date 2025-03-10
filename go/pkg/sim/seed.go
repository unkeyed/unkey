package sim

import (
	"math/rand"
)

// NewSeed generates a new random seed for simulation reproducibility.
// When a simulation fails, the seed can be used to reproduce the exact
// sequence of events that led to the failure.
//
// Example:
//
//	seed := sim.NewSeed()
//	fmt.Printf("Using seed: %d\n", seed)
//
//	// Later, to reproduce a failing test:
//	sim := sim.New[MyState](t, sim.WithSeed(seed))
func NewSeed() int64 {
	// nolint:gosec
	return rand.Int63()
}

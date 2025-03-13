package sim

import "math/rand/v2"

type Rand = rand.Rand

// source implements rand.Source interface with a fixed seed.
// This allows for reproducible random number generation in simulations.
type source struct {
	seed uint64
}

var _ rand.Source = (*source)(nil)

// Uint64 returns the seed as a fixed random number.
// This implementation always returns the same value (the seed),
// which is useful for testing edge cases with specific values.
func (s source) Uint64() uint64 {
	return s.seed
}

package sim

import (
	"math/rand"
)

func NewSeed() int64 {

	// nolint:gosec
	return rand.Int63()
}

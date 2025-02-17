package sim

import (
	"math/rand"
)

func NewSeed() int64 {

	return rand.Int63()
}
